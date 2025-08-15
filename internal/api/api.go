package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/rocketseat-education/semana-tech-go-react-server/internal/logger"
	"github.com/rocketseat-education/semana-tech-go-react-server/internal/store/pgstore"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
)

type apiHandler struct {
	q           *pgstore.Queries
	r           *chi.Mux
	upgrader    websocket.Upgrader
	subscribers map[string]map[*websocket.Conn]context.CancelFunc
	mu          *sync.Mutex
}

func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.r.ServeHTTP(w, r)
}

func NewHandler(q *pgstore.Queries) http.Handler {
	a := apiHandler{
		q:           q,
		upgrader:    websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		subscribers: make(map[string]map[*websocket.Conn]context.CancelFunc),
		mu:          &sync.Mutex{},
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/subscribe/{room_id}", a.handleSubscribe)

	r.Route("/api", func(r chi.Router) {
		r.Route("/rooms", func(r chi.Router) {
			r.Post("/", a.handleCreateRoom)
			r.Get("/", a.handleGetRooms)

			r.Route("/{room_id}", func(r chi.Router) {
				r.Get("/", a.handleGetRoom)

				r.Route("/messages", func(r chi.Router) {
					r.Post("/", a.handleCreateRoomMessage)
					r.Get("/", a.handleGetRoomMessages)

					r.Route("/{message_id}", func(r chi.Router) {
						r.Get("/", a.handleGetRoomMessage)
						r.Patch("/react", a.handleReactToMessage)
						r.Delete("/react", a.handleRemoveReactFromMessage)
						r.Patch("/answer", a.handleMarkMessageAsAnswered)
					})
				})
			})
		})
	})

	a.r = r
	return a
}

const (
	MessageKindMessageCreated          = "message_created"
	MessageKindMessageRactionIncreased = "message_reaction_increased"
	MessageKindMessageRactionDecreased = "message_reaction_decreased"
	MessageKindMessageAnswered         = "message_answered"
)

type MessageMessageReactionIncreased struct {
	ID    string `json:"id"`
	Count int64  `json:"count"`
}

type MessageMessageReactionDecreased struct {
	ID    string `json:"id"`
	Count int64  `json:"count"`
}

type MessageMessageAnswered struct {
	ID string `json:"id"`
}

type MessageMessageCreated struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type Message struct {
	Kind   string `json:"kind"`
	Value  any    `json:"value"`
	RoomID string `json:"-"`
}

func (h apiHandler) notifyClients(msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscribers, ok := h.subscribers[msg.RoomID]
	if !ok || len(subscribers) == 0 {
		logger.Default.Debug(context.Background(), "no subscribers for room", "room_id", msg.RoomID, "message_kind", msg.Kind)
		return
	}

	logger.Default.Debug(context.Background(), "notifying clients", "room_id", msg.RoomID, "message_kind", msg.Kind, "subscriber_count", len(subscribers))

	disconnectedClients := 0
	for conn, cancel := range subscribers {
		if err := conn.WriteJSON(msg); err != nil {
			logger.Default.Error(context.Background(), "failed to send message to client", "room_id", msg.RoomID, "message_kind", msg.Kind, "error", err)
			cancel()
			disconnectedClients++
		}
	}

	if disconnectedClients > 0 {
		logger.Default.Warn(context.Background(), "some clients disconnected", "room_id", msg.RoomID, "disconnected_count", disconnectedClients)
	}
}

func (h apiHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	logger.Default.Info(r.Context(), "WebSocket connection attempt", "room_id", rawRoomID, "client_ip", r.RemoteAddr)

	c, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Default.Warn(r.Context(), "failed to upgrade connection", "room_id", rawRoomID, "client_ip", r.RemoteAddr, "error", err)
		http.Error(w, "failed to upgrade to ws connection", http.StatusBadRequest)
		return
	}

	defer c.Close()

	ctx, cancel := context.WithCancel(r.Context())

	h.mu.Lock()
	if _, ok := h.subscribers[rawRoomID]; !ok {
		h.subscribers[rawRoomID] = make(map[*websocket.Conn]context.CancelFunc)
	}
	logger.Default.Info(r.Context(), "new client connected", "room_id", rawRoomID, "client_ip", r.RemoteAddr, "total_subscribers", len(h.subscribers[rawRoomID])+1)
	h.subscribers[rawRoomID][c] = cancel
	h.mu.Unlock()

	<-ctx.Done()

	h.mu.Lock()
	delete(h.subscribers[rawRoomID], c)
	remainingSubscribers := len(h.subscribers[rawRoomID])
	h.mu.Unlock()

	logger.Default.Info(context.Background(), "client disconnected", "room_id", rawRoomID, "client_ip", r.RemoteAddr, "remaining_subscribers", remainingSubscribers)
}

func (h apiHandler) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	logger.Default.Info(r.Context(), "creating new room")

	type _body struct {
		Theme string `json:"theme"`
	}
	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logger.Default.Warn(r.Context(), "invalid JSON in create room request", "error", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	logger.Default.Debug(r.Context(), "creating room with theme", "theme", body.Theme)

	roomID, err := h.q.InsertRoom(r.Context(), body.Theme)
	if err != nil {
		logger.Default.Error(r.Context(), "failed to insert room", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	logger.Default.Info(r.Context(), "room created successfully", "room_id", roomID.String())

	type response struct {
		ID string `json:"id"`
	}

	sendJSON(w, response{ID: roomID.String()})
}

func (h apiHandler) handleGetRooms(w http.ResponseWriter, r *http.Request) {
	logger.Default.Debug(r.Context(), "fetching all rooms")

	rooms, err := h.q.GetRooms(r.Context())
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		logger.Default.Error(r.Context(), "failed to get rooms", "error", err)
		return
	}

	if rooms == nil {
		rooms = []pgstore.Room{}
	}

	logger.Default.Debug(r.Context(), "rooms fetched successfully", "count", len(rooms))
	sendJSON(w, rooms)
}

func (h apiHandler) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	room, rawRoomID, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	logger.Default.Debug(r.Context(), "fetching room details", "room_id", rawRoomID)
	sendJSON(w, room)
}

func (h apiHandler) handleCreateRoomMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, roomID, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	logger.Default.Info(r.Context(), "creating new message", "room_id", rawRoomID)

	type _body struct {
		Message string `json:"message"`
	}
	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		logger.Default.Warn(r.Context(), "invalid JSON in create message request", "error", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	logger.Default.Debug(r.Context(), "creating message", "room_id", rawRoomID, "message_length", len(body.Message))

	messageID, err := h.q.InsertMessage(r.Context(), pgstore.InsertMessageParams{RoomID: roomID, Message: body.Message})
	if err != nil {
		logger.Default.Error(r.Context(), "failed to insert message", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	logger.Default.Info(r.Context(), "message created successfully", "room_id", rawRoomID, "message_id", messageID.String())

	type response struct {
		ID string `json:"id"`
	}

	sendJSON(w, response{ID: messageID.String()})

	go h.notifyClients(Message{
		Kind:   MessageKindMessageCreated,
		RoomID: rawRoomID,
		Value: MessageMessageCreated{
			ID:      messageID.String(),
			Message: body.Message,
		},
	})
}

func (h apiHandler) handleGetRoomMessages(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, roomID, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	logger.Default.Debug(r.Context(), "fetching messages for room", "room_id", rawRoomID)

	messages, err := h.q.GetRoomMessages(r.Context(), roomID)
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		logger.Default.Error(r.Context(), "failed to get room messages", "room_id", rawRoomID, "error", err)
		return
	}

	if messages == nil {
		messages = []pgstore.Message{}
	}

	logger.Default.Debug(r.Context(), "messages fetched successfully", "room_id", rawRoomID, "count", len(messages))
	sendJSON(w, messages)
}

func (h apiHandler) handleGetRoomMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	rawMessageID := chi.URLParam(r, "message_id")
	messageID, err := uuid.Parse(rawMessageID)
	if err != nil {
		logger.Default.Warn(r.Context(), "invalid message ID in get message request", "room_id", rawRoomID, "message_id", rawMessageID, "error", err)
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	logger.Default.Debug(r.Context(), "fetching specific message", "room_id", rawRoomID, "message_id", rawMessageID)

	messages, err := h.q.GetMessage(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Default.Warn(r.Context(), "message not found", "room_id", rawRoomID, "message_id", rawMessageID)
			http.Error(w, "message not found", http.StatusBadRequest)
			return
		}

		logger.Default.Error(r.Context(), "failed to get message", "room_id", rawRoomID, "message_id", rawMessageID, "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	logger.Default.Debug(r.Context(), "message fetched successfully", "room_id", rawRoomID, "message_id", rawMessageID)
	sendJSON(w, messages)
}

func (h apiHandler) handleReactToMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	rawID := chi.URLParam(r, "message_id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		logger.Default.Warn(r.Context(), "invalid message ID in react request", "message_id", rawID, "error", err)
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	logger.Default.Debug(r.Context(), "adding reaction to message", "room_id", rawRoomID, "message_id", rawID)

	count, err := h.q.ReactToMessage(r.Context(), id)
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		logger.Default.Error(r.Context(), "failed to react to message", "error", err)
		return
	}

	logger.Default.Info(r.Context(), "reaction added successfully", "room_id", rawRoomID, "message_id", rawID, "new_count", count)

	type response struct {
		Count int64 `json:"count"`
	}

	sendJSON(w, response{Count: count})

	go h.notifyClients(Message{
		Kind:   MessageKindMessageRactionIncreased,
		RoomID: rawRoomID,
		Value: MessageMessageReactionIncreased{
			ID:    rawID,
			Count: count,
		},
	})
}

func (h apiHandler) handleRemoveReactFromMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	rawID := chi.URLParam(r, "message_id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		logger.Default.Warn(r.Context(), "invalid message ID in remove reaction request", "message_id", rawID, "error", err)
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	logger.Default.Debug(r.Context(), "removing reaction from message", "room_id", rawRoomID, "message_id", rawID)

	count, err := h.q.RemoveReactionFromMessage(r.Context(), id)
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		logger.Default.Error(r.Context(), "failed to remove reaction from message", "room_id", rawRoomID, "message_id", rawID, "error", err)
		return
	}

	logger.Default.Info(r.Context(), "reaction removed successfully", "room_id", rawRoomID, "message_id", rawID, "new_count", count)

	type response struct {
		Count int64 `json:"count"`
	}

	sendJSON(w, response{Count: count})

	go h.notifyClients(Message{
		Kind:   MessageKindMessageRactionDecreased,
		RoomID: rawRoomID,
		Value: MessageMessageReactionDecreased{
			ID:    rawID,
			Count: count,
		},
	})
}

func (h apiHandler) handleMarkMessageAsAnswered(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, _, ok := h.readRoom(w, r)
	if !ok {
		return
	}

	rawID := chi.URLParam(r, "message_id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		logger.Default.Warn(r.Context(), "invalid message ID in mark answered request", "message_id", rawID, "error", err)
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	logger.Default.Info(r.Context(), "marking message as answered", "room_id", rawRoomID, "message_id", rawID)

	err = h.q.MarkMessageAsAnswered(r.Context(), id)
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		logger.Default.Error(r.Context(), "failed to mark message as answered", "room_id", rawRoomID, "message_id", rawID, "error", err)
		return
	}

	logger.Default.Info(r.Context(), "message marked as answered successfully", "room_id", rawRoomID, "message_id", rawID)

	w.WriteHeader(http.StatusOK)

	go h.notifyClients(Message{
		Kind:   MessageKindMessageAnswered,
		RoomID: rawRoomID,
		Value: MessageMessageAnswered{
			ID: rawID,
		},
	})
}

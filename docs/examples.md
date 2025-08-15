# Exemplos Práticos da API

Este documento contém exemplos práticos de como usar a API de Salas e Mensagens em diferentes cenários.

## 📋 Índice

- [Configuração Inicial](#configuração-inicial)
- [Cenário 1: Criando uma Sala de Chat](#cenário-1-criando-uma-sala-de-chat)
- [Cenário 2: Sistema de Perguntas e Respostas](#cenário-2-sistema-de-perguntas-e-respostas)
- [Cenário 3: Chat em Tempo Real](#cenário-3-chat-em-tempo-real)
- [Cenário 4: Sistema de Votação](#cenário-4-sistema-de-votação)

## ⚙️ Configuração Inicial

### Variáveis de ambiente necessárias
```bash
export API_BASE_URL="http://localhost:8080"
export ROOM_ID="" # Será preenchido após criar uma sala
```

### Testando se a API está rodando
```bash
curl -X GET $API_BASE_URL/api/rooms
```

## 🏠 Cenário 1: Criando uma Sala de Chat

### Passo 1: Criar uma nova sala
```bash
# Criar sala
RESPONSE=$(curl -s -X POST $API_BASE_URL/api/rooms \
  -H "Content-Type: application/json" \
  -d '{
    "theme": "Dúvidas sobre Go Programming"
  }')

# Extrair o ID da sala
ROOM_ID=$(echo $RESPONSE | jq -r '.id')
echo "Sala criada com ID: $ROOM_ID"
```

### Passo 2: Verificar se a sala foi criada
```bash
curl -X GET $API_BASE_URL/api/rooms/$ROOM_ID | jq '.'
```

### Passo 3: Listar todas as salas
```bash
curl -X GET $API_BASE_URL/api/rooms | jq '.'
```

**Resultado esperado:**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "theme": "Dúvidas sobre Go Programming"
  }
]
```

## ❓ Cenário 2: Sistema de Perguntas e Respostas

### Passo 1: Criar várias perguntas
```bash
# Pergunta 1
MESSAGE1=$(curl -s -X POST $API_BASE_URL/api/rooms/$ROOM_ID/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Como funciona o garbage collector em Go?"
  }' | jq -r '.id')

# Pergunta 2
MESSAGE2=$(curl -s -X POST $API_BASE_URL/api/rooms/$ROOM_ID/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Qual é a diferença entre goroutines e threads?"
  }' | jq -r '.id')

# Pergunta 3
MESSAGE3=$(curl -s -X POST $API_BASE_URL/api/rooms/$ROOM_ID/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Como implementar channels de forma eficiente?"
  }' | jq -r '.id')

echo "Mensagens criadas: $MESSAGE1, $MESSAGE2, $MESSAGE3"
```

### Passo 2: Simular votação nas perguntas
```bash
# Adicionar 5 reações na primeira pergunta
for i in {1..5}; do
  curl -s -X PATCH $API_BASE_URL/api/rooms/$ROOM_ID/messages/$MESSAGE1/react
done

# Adicionar 3 reações na segunda pergunta
for i in {1..3}; do
  curl -s -X PATCH $API_BASE_URL/api/rooms/$ROOM_ID/messages/$MESSAGE2/react
done

# Adicionar 7 reações na terceira pergunta
for i in {1..7}; do
  curl -s -X PATCH $API_BASE_URL/api/rooms/$ROOM_ID/messages/$MESSAGE3/react
done
```

### Passo 3: Marcar uma pergunta como respondida
```bash
# Marcar a pergunta mais votada como respondida
curl -X PATCH $API_BASE_URL/api/rooms/$ROOM_ID/messages/$MESSAGE3/answer
```

### Passo 4: Ver o resultado final
```bash
curl -X GET $API_BASE_URL/api/rooms/$ROOM_ID/messages | jq '.'
```

**Resultado esperado:**
```json
[
  {
    "id": "message1-id",
    "room_id": "room-id",
    "message": "Como funciona o garbage collector em Go?",
    "reaction_count": 5,
    "answered": false
  },
  {
    "id": "message2-id", 
    "room_id": "room-id",
    "message": "Qual é a diferença entre goroutines e threads?",
    "reaction_count": 3,
    "answered": false
  },
  {
    "id": "message3-id",
    "room_id": "room-id", 
    "message": "Como implementar channels de forma eficiente?",
    "reaction_count": 7,
    "answered": true
  }
]
```

## 💬 Cenário 3: Chat em Tempo Real

### Cliente JavaScript para WebSocket
Salve este código como `chat-client.html`:

```html
<!DOCTYPE html>
<html>
<head>
    <title>Chat em Tempo Real</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        #messages { height: 400px; overflow-y: scroll; border: 1px solid #ccc; padding: 10px; margin-bottom: 10px; }
        .message { margin-bottom: 10px; padding: 5px; background: #f0f0f0; border-radius: 5px; }
        .message.answered { background: #d4edda; }
        .reactions { color: #007bff; font-weight: bold; }
        input[type="text"] { width: 70%; padding: 5px; }
        button { padding: 5px 10px; margin-left: 5px; }
    </style>
</head>
<body>
    <h1>Chat da Sala: <span id="roomTheme">Carregando...</span></h1>
    
    <div id="messages"></div>
    
    <div>
        <input type="text" id="messageInput" placeholder="Digite sua mensagem..." onkeypress="if(event.key==='Enter') sendMessage()">
        <button onclick="sendMessage()">Enviar</button>
    </div>

    <script>
        const ROOM_ID = prompt("Digite o ID da sala:") || "550e8400-e29b-41d4-a716-446655440000";
        const API_BASE = "http://localhost:8080";
        
        let ws;
        let messages = {};

        // Conectar WebSocket
        function connectWebSocket() {
            ws = new WebSocket(`ws://localhost:8080/subscribe/${ROOM_ID}`);
            
            ws.onopen = function() {
                console.log('Conectado ao WebSocket');
                loadRoomInfo();
                loadMessages();
            };

            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);
                handleWebSocketMessage(data);
            };

            ws.onclose = function() {
                console.log('Desconectado do WebSocket');
                setTimeout(connectWebSocket, 3000); // Reconectar após 3 segundos
            };

            ws.onerror = function(error) {
                console.error('Erro no WebSocket:', error);
            };
        }

        // Processar mensagens do WebSocket
        function handleWebSocketMessage(data) {
            switch(data.kind) {
                case 'message_created':
                    addMessageToUI({
                        id: data.value.id,
                        message: data.value.message,
                        reaction_count: 0,
                        answered: false
                    });
                    break;
                
                case 'message_reaction_increased':
                    updateMessageReactions(data.value.id, data.value.count);
                    break;
                
                case 'message_reaction_decreased':
                    updateMessageReactions(data.value.id, data.value.count);
                    break;
                
                case 'message_answered':
                    markMessageAsAnswered(data.value.id);
                    break;
            }
        }

        // Carregar informações da sala
        async function loadRoomInfo() {
            try {
                const response = await fetch(`${API_BASE}/api/rooms/${ROOM_ID}`);
                const room = await response.json();
                document.getElementById('roomTheme').textContent = room.theme;
            } catch (error) {
                console.error('Erro ao carregar sala:', error);
                document.getElementById('roomTheme').textContent = 'Erro ao carregar';
            }
        }

        // Carregar mensagens existentes
        async function loadMessages() {
            try {
                const response = await fetch(`${API_BASE}/api/rooms/${ROOM_ID}/messages`);
                const messagesData = await response.json();
                
                const messagesDiv = document.getElementById('messages');
                messagesDiv.innerHTML = '';
                
                messagesData.forEach(message => {
                    addMessageToUI(message);
                });
            } catch (error) {
                console.error('Erro ao carregar mensagens:', error);
            }
        }

        // Adicionar mensagem na interface
        function addMessageToUI(message) {
            messages[message.id] = message;
            
            const messagesDiv = document.getElementById('messages');
            const messageDiv = document.createElement('div');
            messageDiv.className = `message ${message.answered ? 'answered' : ''}`;
            messageDiv.id = `message-${message.id}`;
            messageDiv.innerHTML = `
                <div>${message.message}</div>
                <div class="reactions">
                    👍 <span id="count-${message.id}">${message.reaction_count}</span>
                    <button onclick="reactToMessage('${message.id}')" style="margin-left: 10px;">👍</button>
                    <button onclick="removeReaction('${message.id}')" style="margin-left: 5px;">👎</button>
                    ${!message.answered ? `<button onclick="markAsAnswered('${message.id}')" style="margin-left: 10px;">✅ Marcar como respondida</button>` : '<span style="color: green;">✅ Respondida</span>'}
                </div>
            `;
            
            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        // Enviar nova mensagem
        async function sendMessage() {
            const input = document.getElementById('messageInput');
            const message = input.value.trim();
            
            if (!message) return;
            
            try {
                await fetch(`${API_BASE}/api/rooms/${ROOM_ID}/messages`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ message })
                });
                
                input.value = '';
            } catch (error) {
                console.error('Erro ao enviar mensagem:', error);
            }
        }

        // Reagir a mensagem
        async function reactToMessage(messageId) {
            try {
                await fetch(`${API_BASE}/api/rooms/${ROOM_ID}/messages/${messageId}/react`, {
                    method: 'PATCH'
                });
            } catch (error) {
                console.error('Erro ao reagir:', error);
            }
        }

        // Remover reação
        async function removeReaction(messageId) {
            try {
                await fetch(`${API_BASE}/api/rooms/${ROOM_ID}/messages/${messageId}/react`, {
                    method: 'DELETE'
                });
            } catch (error) {
                console.error('Erro ao remover reação:', error);
            }
        }

        // Marcar como respondida
        async function markAsAnswered(messageId) {
            try {
                await fetch(`${API_BASE}/api/rooms/${ROOM_ID}/messages/${messageId}/answer`, {
                    method: 'PATCH'
                });
            } catch (error) {
                console.error('Erro ao marcar como respondida:', error);
            }
        }

        // Atualizar contadores de reação
        function updateMessageReactions(messageId, count) {
            const countElement = document.getElementById(`count-${messageId}`);
            if (countElement) {
                countElement.textContent = count;
            }
        }

        // Marcar mensagem como respondida na UI
        function markMessageAsAnswered(messageId) {
            const messageElement = document.getElementById(`message-${messageId}`);
            if (messageElement) {
                messageElement.className = 'message answered';
                // Atualizar botões para mostrar status de respondida
                loadMessages(); // Recarregar para atualizar a interface
            }
        }

        // Iniciar aplicação
        connectWebSocket();
    </script>
</body>
</html>
```

### Como usar o cliente:
1. Abra o arquivo `chat-client.html` no navegador
2. Digite o ID da sala criada anteriormente
3. Teste enviando mensagens, reagindo e marcando como respondidas

## 🗳️ Cenário 4: Sistema de Votação

### Script para simular votação em massa
Salve como `voting-simulation.sh`:

```bash
#!/bin/bash

API_BASE="http://localhost:8080"
ROOM_ID="$1"

if [ -z "$ROOM_ID" ]; then
    echo "Uso: $0 <ROOM_ID>"
    exit 1
fi

echo "🗳️  Simulando sistema de votação na sala: $ROOM_ID"

# Criar várias propostas
proposals=(
    "Implementar autenticação JWT na API"
    "Adicionar sistema de cache Redis" 
    "Migrar para Docker containers"
    "Implementar rate limiting"
    "Adicionar logs estruturados"
    "Criar testes de integração"
    "Implementar CI/CD com GitHub Actions"
    "Adicionar documentação Swagger"
)

echo "📝 Criando propostas..."
proposal_ids=()

for proposal in "${proposals[@]}"; do
    response=$(curl -s -X POST $API_BASE/api/rooms/$ROOM_ID/messages \
        -H "Content-Type: application/json" \
        -d "{\"message\": \"$proposal\"}")
    
    id=$(echo $response | jq -r '.id')
    proposal_ids+=($id)
    echo "  ✅ Criada: $proposal (ID: $id)"
done

echo ""
echo "🗳️  Simulando votação (cada proposta recebe entre 1-15 votos)..."

for id in "${proposal_ids[@]}"; do
    # Gerar número aleatório de votos entre 1 e 15
    votes=$((RANDOM % 15 + 1))
    
    for ((i=1; i<=votes; i++)); do
        curl -s -X PATCH $API_BASE/api/rooms/$ROOM_ID/messages/$id/react > /dev/null
    done
    
    echo "  📊 Proposta $id recebeu $votes votos"
done

echo ""
echo "📊 Resultado final da votação:"
echo "================================"

# Buscar todas as mensagens e ordenar por votos
messages=$(curl -s -X GET $API_BASE/api/rooms/$ROOM_ID/messages)

echo "$messages" | jq -r '.[] | "\(.reaction_count) votos - \(.message)"' | sort -nr

echo ""
echo "🏆 Proposta mais votada:"
winner=$(echo "$messages" | jq -r 'max_by(.reaction_count)')
winner_id=$(echo "$winner" | jq -r '.id')
winner_message=$(echo "$winner" | jq -r '.message')
winner_votes=$(echo "$winner" | jq -r '.reaction_count')

echo "   $winner_message ($winner_votes votos)"

# Marcar a proposta vencedora como "respondida" (implementada)
echo ""
echo "✅ Marcando proposta vencedora como implementada..."
curl -s -X PATCH $API_BASE/api/rooms/$ROOM_ID/messages/$winner_id/answer

echo "✅ Simulação concluída!"
```

### Como executar:
```bash
chmod +x voting-simulation.sh
./voting-simulation.sh $ROOM_ID
```

## 📊 Monitoramento em Tempo Real

### Script para monitorar atividade
Salve como `monitor.sh`:

```bash
#!/bin/bash

API_BASE="http://localhost:8080"
ROOM_ID="$1"

if [ -z "$ROOM_ID" ]; then
    echo "Uso: $0 <ROOM_ID>"
    exit 1
fi

echo "📊 Monitorando sala: $ROOM_ID"
echo "================================"

while true; do
    clear
    echo "📊 Status da Sala em Tempo Real"
    echo "================================"
    
    # Informações da sala
    room_info=$(curl -s -X GET $API_BASE/api/rooms/$ROOM_ID)
    room_theme=$(echo "$room_info" | jq -r '.theme')
    
    echo "🏠 Sala: $room_theme"
    echo "🆔 ID: $ROOM_ID"
    echo ""
    
    # Estatísticas das mensagens
    messages=$(curl -s -X GET $API_BASE/api/rooms/$ROOM_ID/messages)
    
    total_messages=$(echo "$messages" | jq '. | length')
    total_reactions=$(echo "$messages" | jq '[.[].reaction_count] | add // 0')
    answered_messages=$(echo "$messages" | jq '[.[] | select(.answered == true)] | length')
    
    echo "📊 Estatísticas:"
    echo "   💬 Total de mensagens: $total_messages"
    echo "   👍 Total de reações: $total_reactions"
    echo "   ✅ Mensagens respondidas: $answered_messages"
    echo ""
    
    # Top 5 mensagens mais votadas
    echo "🏆 Top 5 Mensagens Mais Votadas:"
    echo "$messages" | jq -r 'sort_by(-.reaction_count) | .[:5] | .[] | "   \(.reaction_count)👍 - \(.message) \(if .answered then "✅" else "" end)"'
    
    echo ""
    echo "⏰ Última atualização: $(date)"
    echo "🔄 Pressione Ctrl+C para parar o monitoramento"
    
    sleep 5
done
```

### Como usar:
```bash
chmod +x monitor.sh
./monitor.sh $ROOM_ID
```

## 🔧 Utilitários

### Limpeza de dados
```bash
# Script para limpar todas as reações de uma sala
cleanup_reactions() {
    local room_id=$1
    messages=$(curl -s -X GET $API_BASE/api/rooms/$room_id/messages)
    
    echo "$messages" | jq -r '.[].id' | while read message_id; do
        # Remover todas as reações (assumindo máximo de 50 reações por mensagem)
        for i in {1..50}; do
            response=$(curl -s -X DELETE $API_BASE/api/rooms/$room_id/messages/$message_id/react)
            count=$(echo "$response" | jq -r '.count // 0')
            
            if [ "$count" = "0" ]; then
                break
            fi
        done
        echo "Limpeza da mensagem $message_id concluída"
    done
}

# Uso: cleanup_reactions $ROOM_ID
```

### Backup de dados
```bash
# Fazer backup de uma sala
backup_room() {
    local room_id=$1
    local backup_file="backup_room_${room_id}_$(date +%Y%m%d_%H%M%S).json"
    
    room_data=$(curl -s -X GET $API_BASE/api/rooms/$room_id)
    messages_data=$(curl -s -X GET $API_BASE/api/rooms/$room_id/messages)
    
    echo "{\"room\": $room_data, \"messages\": $messages_data}" > "$backup_file"
    echo "Backup salvo em: $backup_file"
}

# Uso: backup_room $ROOM_ID
```

Estes exemplos demonstram cenários reais de uso da API, desde a criação básica de salas até sistemas complexos de votação e monitoramento em tempo real.

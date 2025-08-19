package hub

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// --- WebSocket upgrade e handlers ---

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // simples p/ demo
}

// ServeWS lida com o upgrade HTTP -> WebSocket e registra o cliente
func ServeWS(h *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		name = randomName()
	}

	c := &Client{
		hub:  h,
		conn: conn,
		send: make(chan Message, 64),
		name: name,
	}

	h.register <- c

	// Envia uma mensagem de boas-vindas somente para o novo cliente
	welcome := Message{Type: "system", Text: "Bem-vindo, " + name + "!", Time: nowMillis()}
	c.send <- welcome

	go c.writePump()
	go c.readPump()
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

func randomName() string {
	// Nome simples aleatório
	// Em produção, você poderia usar UUIDs ou login real
	return "Guest-" + strings.ToUpper(randString(6))
}

// gerador simples para sufixo do nome
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
		time.Sleep(time.Microsecond) // evita colisões num loop rápido
	}
	return string(b)
}

// Client representa uma conexão WS com seu canal de saída
// Implementações de leitura/escrita ficam em internal/client/client.go
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan Message
	name string
}

// Utilitário para decodificar mensagens recebidas
func decodeMessage(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	if m.Type == "" {
		m.Type = "chat"
	}
	m.Time = nowMillis()
	return m, err
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		m, err := decodeMessage(message)
		if err != nil {
			continue
		}
		if m.Type == "chat" {
			m.From = c.name
		}
		c.hub.broadcast <- m
	}
}

// writePump envia mensagens do Hub para o cliente
func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Canal fechado -> encerra
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			// ping periódico para manter a conexão viva
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type Message struct {
	Type string `json:"type"`
	Text string `json:"text"`
	From string `json:"from,omitempty"`
	Time int64  `json:"time"`
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}
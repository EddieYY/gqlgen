//go:generate gqlgen -out generated.go

package chat

import (
	"time"
)

type Chatroom struct {
	ID       string
	Title    string
	Users    []User
	Messages []Message
}

type User struct {
	ID          string
	DisplayName string
}

type Message struct {
	ID        string
	Text      string
	CreatedBy User
	CreatedAt time.Time
}

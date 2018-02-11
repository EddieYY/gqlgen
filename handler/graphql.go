package handler

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"strings"

	"github.com/gorilla/websocket"
	"github.com/vektah/gqlgen/neelance/errors"
)

type Executor func(ctx context.Context, document string, operationName string, variables map[string]interface{}, w io.Writer) []*errors.QueryError

type errorResponse struct {
	Errors []*errors.QueryError `json:"errors"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func GraphQL(resolver Executor) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var params struct {
			Query         string                 `json:"query"`
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}

		if strings.Contains(r.Header["Upgrade"], "websocket") {
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
		client.hub.register <- client

		// Allow collection of memory referenced by the caller by doing all work in
		// new goroutines.
		go client.writePump()
		go client.readPump()

		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" {
			params.Query = r.URL.Query().Get("query")
			params.OperationName = r.URL.Query().Get("operationName")

			if variables := r.URL.Query().Get("variables"); variables != "" {
				if err := json.Unmarshal([]byte(variables), &params.Variables); err != nil {
					sendError(w, http.StatusBadRequest, []*errors.QueryError{errors.Errorf("variables could not be decoded")})
					return
				}
			}
		} else {
			if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
				sendError(w, http.StatusBadRequest, []*errors.QueryError{errors.Errorf("json body could not be decoded")})
				return
			}
		}

		errs := resolver(r.Context(), params.Query, params.OperationName, params.Variables, w)
		if errs != nil {
			sendError(w, http.StatusUnprocessableEntity, errs)
		}
	})
}

func sendError(w http.ResponseWriter, code int, errs []*errors.QueryError) {
	w.WriteHeader(code)

	b, err := json.Marshal(errorResponse{Errors: errs})
	if err != nil {
		panic(err)
	}
	w.Write(b)
}

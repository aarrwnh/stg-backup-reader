package reader

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 64,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	address  string
	ws       bool
	certPath string
	keyPath  string
)

func init() {
	flag.StringVar(&certPath, "cert", "", "path to SSL/TLS certificate file")
	flag.StringVar(&keyPath, "key", "", "path to SSL/TLS private key file")
	flag.StringVar(&address, "a", "127.0.0.1:50001", "address to use")
	flag.BoolVar(&ws, "ws", false, "use websockets")
}

type MessageIn struct {
	Query string `json:"query"`
	Id    int    `json:"id"`
	Type  string `json:"type"`
}

type MessageOut struct {
	Data any    `json:"data"`
	Id   int    `json:"id"`
	Type string `json:"type"`
}

func NewMessage(app *App, data MessageIn) MessageOut {
	var newData any
	switch data.Type {
	case "count":
		newData = app.size
	case "tabs":
		var tabs []string
		// TODO: add/use open limit?
		for _, t := range app.found {
			tabs = append(tabs, t.Url)
		}
		newData = tabs
	default:
		panic("`MessageIn.Type` is unknown")
	}
	return MessageOut{Data: newData, Id: data.Id, Type: data.Type}
}

func StartWebsocket(app *App) {
	if !ws {
		return
	}
	http.HandleFunc("/ws", wsHandler(app))

	if certPath == "" || keyPath == "" {
		log.Fatal("Warning: SSL/TLS certificate and/or private key file not provided.")
	} else {
		err := http.ListenAndServeTLS(address, expandHome(certPath), expandHome(keyPath), nil)
		if err != nil {
			panic(err)
		}
	}
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		dirname, _ := os.UserHomeDir()
		p = filepath.Join(dirname, p[2:])
	}
	return p
}

func echo(conn *websocket.Conn, app *App) {
	defer func() { // cleanup
		conn.Close()
		app.wsConnected = false
		app.UpdateTitle()
	}()

	app.wsConnected = true
	app.UpdateTitle()

	for {
		time.Sleep(time.Millisecond * 500)

		// messageType, p, err := conn.ReadMessage()
		// if err != nil {
		// 	if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) { }
		// 	break
		// }

		var data MessageIn
		if err := conn.ReadJSON(&data); err != nil {
			break
		}

		if data.Query != "" {
			fmt.Printf("%s\n", data.Query)

			app.FindTabs(data.Query, false)
			msg := NewMessage(app, data)
			if err := conn.WriteJSON(msg); err != nil {
				log.Println(err)
				break
			}

			app.UpdateTitle()

			// cleanup
			switch data.Type {
			case "tabs":
				app.ForceRemove()
			}

			fmt.Print("\n> ")
		}
	}
}

type HTTPHandler func(w http.ResponseWriter, r *http.Request)

func wsHandler(app *App) HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		// conn.SetReadLimit(maxMessageSize)
		// conn.SetReadDeadline(time.Now().Add(pongWait))
		// conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

		// tx := make(chan interface{})
		go echo(conn, app)
		// go responder(tx)
	}
}

package main

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
	WriteBufferSize: 512,
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
}

type MessageOut struct {
	Count int `json:"count"`
	Id    int `json:"id"`
}

func startWs(app *App) {
	if !ws {
		return
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(app, w, r)
	})

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
	path := p
	if strings.HasPrefix(p, "~/") {
		dirname, _ := os.UserHomeDir()
		path = filepath.Join(dirname, p[2:])
	}
	return path
}

// type Conn struct {
// 	conn *websocket.Conn
// }
// func (c Conn) ReadMessage() (messageType int, p []byte, err error) { }

func changeTitle(t string) {
	fmt.Fprintf(os.Stdout, "\033]0;%s\007", t)
}

func echo(conn *websocket.Conn, app *App) {
	defer func() {
		conn.Close()
		// TODO: reset title after disconnect
		changeTitle("")
	}()

	var rxc, txc int

	changeTitle("* connected")

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

		rxc += 1
		if data.Query != "" { // != "" does not work?
			fmt.Printf("%s\n", data.Query)
			app.FindTabs(data.Query, true)
			fmt.Print("\n> ")

			msg := MessageOut{Count: len(app.found), Id: data.Id}
			if err := conn.WriteJSON(msg); err != nil {
				log.Println(err)
				break
			}

			txc += 1
		}

		changeTitle(fmt.Sprintf("* [rx:%d tx:%d]", rxc, txc))
	}
}

func wsHandler(app *App, w http.ResponseWriter, r *http.Request) {
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

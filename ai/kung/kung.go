// Kung is a proof of concept AI that interfaces with the Pao server
// but doesn't make any meaningful game decisions
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/arbrown/pao/game/command"
	"github.com/arbrown/pao/game/util"
	"github.com/gorilla/websocket"
)

type kung struct {
	conn *websocket.Conn
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	fmt.Println("Hello, Kung")
	host, port := os.Getenv("KUNGHOST"), os.Getenv("KUNGPORT")
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "2021"
	}
	fmt.Printf("%v:%v", host, port)
	bind := fmt.Sprintf("%v:%v", host, port)
	http.HandleFunc("/", kungServe)
	http.ListenAndServe(bind, nil)
}

func kungServe(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Request: %v\n", r)
	conn, err := upgrader.Upgrade(w, r, nil)
	fmt.Printf("Got a new customer!\n")
	if err != nil {
		fmt.Printf("kungErr = %v\n", err.Error())
		return
	}
	f := kung{
		conn: conn,
	}
	go playKung(f)
	fmt.Printf("%v - Exiting Handler\n", time.Now())
}

func playKung(kung kung) {

	defer readLoop(kung.conn)
	for {
		var com command.Command
		_, bytes, err := kung.conn.ReadMessage()
		if err != nil {
			fmt.Printf("%v - Error reading from websocket: %v\n", time.Now(), err.Error())
			return
		}
		if err = json.Unmarshal(bytes, &com); err != nil {
			fmt.Printf("Error decoding websocket message: %v\n", bytes)
		}
		switch com.Action {
		case "board":
			var bc command.BoardCommand
			if err = json.Unmarshal(bytes, &bc); err != nil {
				fmt.Printf("Error decoding board command: %v\n", string(bytes))
			}
			kung.processBoard(bc)
		default:
			fmt.Printf("Message: %v\n", string(bytes))
		}
	}
}

func (f *kung) processBoard(bc command.BoardCommand) {
	gs := util.ParseGameState(bc)
	if !bc.YourTurn {
		return
	}
	if len(gs.RemainingPieces) == 0 {
		f.resign()
		return
	}
	for {
		rank, file := rand.Intn(len(gs.KnownBoard)), rand.Intn(len(gs.KnownBoard[0]))
		piece := gs.KnownBoard[rank][file]
		if piece == "?" {
			f.flip(rank, file)
			return
		}
	}
}

func (f *kung) flip(rank, file int) {
	s := "?" + util.ToNotation(rank, file)
	com := command.Command{
		Action:   "move",
		Argument: s,
	}
	f.sendCommand(com)
}

func (f *kung) resign() {
	com := command.Command{
		Action: "resign",
	}
	f.sendCommand(com)
}

func (f *kung) sendCommand(c command.Command) {
	if err := f.conn.WriteJSON(c); err != nil {
		fmt.Printf("Error sending command: %v", err.Error())
	}
}

func readLoop(c *websocket.Conn) {
	fmt.Printf("%v - Closing conn\n", time.Now())
	for {
		if _, _, err := c.NextReader(); err != nil {
			c.Close()
			break
		}
	}
}

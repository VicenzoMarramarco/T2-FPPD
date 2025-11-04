// server.go
package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"

	"jogo/common/shared"
)

// Servidor RPC
type GameServer struct {
	mu sync.Mutex

	players map[string]shared.PlayerState // clientID -> PlayerState
	lastSeq map[string]uint64             // clientID -> last applied sequence number
	names   map[string]string             // clientID -> name
	nextID  uint64

	mapLines []string // authoritative map as lines
}

// loadMapLines loads a text map file into a slice of strings.
func loadMapLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, er := f.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			return nil, er
		}
	}
	// split on \n preserving contents without trailing \r
	lines := []string{}
	start := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] == '\n' {
			line := string(buf[start:i])
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start <= len(buf)-1 {
		line := string(buf[start:])
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func NewGameServer() *GameServer {
	gs := &GameServer{
		players:  make(map[string]shared.PlayerState),
		lastSeq:  make(map[string]uint64),
		names:    make(map[string]string),
		nextID:   1,
		mapLines: nil,
	}
	// Try to load map from local file (mapa.txt); non-fatal if missing
	if lines, err := loadMapLines("mapa.txt"); err == nil {
		gs.mapLines = lines
	}
	return gs
}

// Register: client pede um clientID
func (gs *GameServer) Register(args shared.RegisterArgs, reply *shared.RegisterReply) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	id := fmt.Sprintf("C%06d", gs.nextID)
	gs.nextID++
	gs.names[id] = args.Name
	gs.players[id] = shared.PlayerState{ID: id, Name: args.Name, X: 0, Y: 0}
	gs.lastSeq[id] = 0

	reply.ClientID = id
	fmt.Printf("[SERVER] Register request: name=%s -> clientID=%s\n", args.Name, id)
	return nil
}

// SendCommand: cliente envia comando (com sequenceNumber)
func (gs *GameServer) SendCommand(cmd shared.Command, reply *shared.CommandReply) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	fmt.Printf("[SERVER] Got Command from %s seq=%d pos=(%d,%d) cmd=%s\n",
		cmd.ClientID, cmd.Sequence, cmd.ReportedX, cmd.ReportedY, cmd.CommandString)

	if _, ok := gs.players[cmd.ClientID]; !ok {
		reply.Applied = false
		reply.Error = "unknown client"
		return errors.New(reply.Error)
	}

	last := gs.lastSeq[cmd.ClientID]
	if cmd.Sequence <= last {
		reply.Applied = false
		reply.Error = "duplicate or old sequence"
		fmt.Printf("[SERVER] Duplicate/old command ignored: client=%s seq=%d last=%d\n", cmd.ClientID, cmd.Sequence, last)
		return nil
	}

	ps := gs.players[cmd.ClientID]
	ps.X = cmd.ReportedX
	ps.Y = cmd.ReportedY
	gs.players[cmd.ClientID] = ps
	gs.lastSeq[cmd.ClientID] = cmd.Sequence

	reply.Applied = true
	reply.Error = ""
	fmt.Printf("[SERVER] Applied command: client=%s newpos=(%d,%d) seq=%d\n", cmd.ClientID, ps.X, ps.Y, cmd.Sequence)
	return nil
}

// GetState: cliente pede estado atual do jogo
func (gs *GameServer) GetState(args shared.GetStateArgs, reply *shared.GameState) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	players := make([]shared.PlayerState, 0, len(gs.players))
	for _, p := range gs.players {
		players = append(players, p)
	}
	reply.Players = players
	reply.Time = time.Now()
	reply.MapLines = gs.mapLines

	fmt.Printf("[SERVER] GetState requested by %s -> %d players returned\n", args.ClientID, len(players))
	return nil
}

// StartRPCServer starts the RPC server on the given address and returns the listener.
func StartRPCServer(addr string) (net.Listener, error) {
	gs := NewGameServer()
	if err := rpc.Register(gs); err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[SERVER] RPC server listening on %s\n", addr)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// listener likely closed
				fmt.Printf("[SERVER] accept error: %v\n", err)
				return
			}
			go rpc.ServeConn(conn)
		}
	}()

	return listener, nil
}

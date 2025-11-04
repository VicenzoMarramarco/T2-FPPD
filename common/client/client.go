// client.go
package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"strconv"
	"strings"
	"sync"
	"time"

	"jogo/common/shared"
)

// ---- Cliente ----
type Client struct {
	rpcClient *rpc.Client
	name      string
	clientID  string

	mu sync.Mutex

	x, y int
	seq  uint64

	// local state broadcaster
	subsMu  sync.Mutex
	subs    map[net.Conn]struct{}
	stateLn net.Listener
}

func NewClient(name string, rpcAddr string) (*Client, error) {
	c := &Client{name: name, x: 0, y: 0, seq: 0, subs: make(map[net.Conn]struct{})}
	conn, err := rpc.Dial("tcp", rpcAddr)
	if err != nil {
		return nil, err
	}
	c.rpcClient = conn

	var rr shared.RegisterReply
	err = c.rpcClient.Call("GameServer.Register", shared.RegisterArgs{Name: name}, &rr)
	if err != nil {
		return nil, err
	}
	c.clientID = rr.ClientID
	fmt.Printf("[CLIENT %s] Registered with id=%s\n", name, c.clientID)
	return c, nil
}

// sendCommandWithRetry (com backoff simples)
func (c *Client) sendCommandWithRetry(cmd shared.Command) (shared.CommandReply, error) {
	var lastErr error
	var rep shared.CommandReply

	for attempt := 1; attempt <= 6; attempt++ {
		fmt.Printf("[CLIENT %s] Sending command seq=%d attempt=%d pos=(%d,%d)\n",
			c.name, cmd.Sequence, attempt, cmd.ReportedX, cmd.ReportedY)

		callErr := c.rpcClient.Call("GameServer.SendCommand", cmd, &rep)
		if callErr == nil {
			return rep, nil
		}
		lastErr = callErr
		fmt.Printf("[CLIENT %s] RPC error: %v. Retrying...\n", c.name, callErr)
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
	}
	return rep, lastErr
}

// polling do estado do servidor
func (c *Client) StartPolling() {
	go func() {
		for {
			var gs shared.GameState
			err := c.rpcClient.Call("GameServer.GetState", shared.GetStateArgs{ClientID: c.clientID}, &gs)
			if err != nil {
				fmt.Printf("[CLIENT %s] GetState error: %v\n", c.name, err)
			} else {
				fmt.Printf("\n[CLIENT %s] GetState: %d players at %s\n", c.name, len(gs.Players), gs.Time.Format("15:04:05"))
				for _, p := range gs.Players {
					fmt.Printf("   -> %s (%s): (%d,%d)\n", p.ID, p.Name, p.X, p.Y)
				}
				// broadcast to local UI listeners
				c.broadcastState(gs)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
}

// listener local para receber comandos do jogo (jogo.exe/jogo.go)
func (c *Client) StartLocalCommandListener(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("[CLIENT %s] Local command listener running on %s\n", c.name, addr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Printf("[CLIENT] accept error: %v\n", err)
				continue
			}
			go c.handleLocalConn(conn)
		}
	}()
	return nil
}

func (c *Client) handleLocalConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Espera formato: MOVE <x> <y>
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		switch strings.ToUpper(parts[0]) {
		case "MOVE":
			if len(parts) < 3 {
				// protocolo local sem resposta
				continue
			}
			x, err1 := strconv.Atoi(parts[1])
			y, err2 := strconv.Atoi(parts[2])
			if err1 != nil || err2 != nil {
				continue
			}
			// envia comando ao servidor
			c.mu.Lock()
			c.x = x
			c.y = y
			c.seq++
			seq := c.seq
			c.mu.Unlock()

			cmd := shared.Command{
				ClientID:      c.clientID,
				Sequence:      seq,
				ReportedX:     x,
				ReportedY:     y,
				CommandString: "MOVE",
			}
			// Handshake: responde imediatamente para permitir o jogo ler e fechar sem reset
			if _, err := conn.Write([]byte("OK\n")); err != nil {
				// ignorar erro de escrita se o outro lado fechar antes
			}
			_, _ = c.sendCommandWithRetry(cmd)
		default:
			// comando desconhecido: ignorar
		}
	}
	if err := scanner.Err(); err != nil {
		// Em Windows, quando o lado remoto fecha logo após enviar, pode vir wsarecv/aborted.
		es := err.Error()
		if strings.Contains(strings.ToLower(es), "wsarecv") ||
			strings.Contains(strings.ToLower(es), "aborted") ||
			strings.Contains(strings.ToLower(es), "reset") ||
			strings.Contains(strings.ToLower(es), "closed") {
			// tratar como fechamento normal
			return
		}
		fmt.Printf("[CLIENT] local conn error: %v\n", err)
	}
}

// ID returns this client's server-assigned id
func (c *Client) ID() string { return c.clientID }

// --- Integration helper: report positions from a shared channel ---
func StartPositionReporter(posCh <-chan [2]int, clientID string, serverAddr string, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case pos := <-posCh:
				conn, err := rpc.Dial("tcp", serverAddr)
				if err != nil {
					log.Println("Erro ao conectar servidor:", err)
					continue
				}
				var reply shared.CommandReply
				cmd := shared.Command{
					ClientID:      clientID,
					Sequence:      uint64(time.Now().UnixNano()),
					ReportedX:     pos[0],
					ReportedY:     pos[1],
					CommandString: "UPDATE_POSITION",
				}
				// chamada correta ao servidor
				err = conn.Call("GameServer.SendCommand", cmd, &reply)
				if err != nil {
					log.Println("Erro RPC:", err)
				}
				conn.Close()
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// StartLocalStateBroadcaster starts a TCP server to stream game state locally to the UI.
func (c *Client) StartLocalStateBroadcaster(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	c.stateLn = ln
	fmt.Printf("[CLIENT %s] Local state broadcaster on %s\n", c.name, addr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			c.subsMu.Lock()
			c.subs[conn] = struct{}{}
			c.subsMu.Unlock()
			go c.handleSub(conn)
		}
	}()
	return nil
}

func (c *Client) handleSub(conn net.Conn) {
	defer func() {
		c.subsMu.Lock()
		delete(c.subs, conn)
		c.subsMu.Unlock()
		conn.Close()
	}()
	// keep the connection open until closed by peer
	buf := make([]byte, 1)
	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		if _, err := conn.Read(buf); err != nil {
			return
		}
	}
}

func (c *Client) broadcastState(gs shared.GameState) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	if len(c.subs) == 0 {
		return
	}
	// Build message
	var b strings.Builder
	fmt.Fprintf(&b, "SELF %s\n", c.clientID)
	if n := len(gs.MapLines); n > 0 {
		fmt.Fprintf(&b, "MAP %d\n", n)
		for _, line := range gs.MapLines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	} else {
		b.WriteString("MAP 0\n")
	}
	fmt.Fprintf(&b, "PLAYERS %d\n", len(gs.Players))
	for _, p := range gs.Players {
		fmt.Fprintf(&b, "%s\t%s\t%d\t%d\n", p.ID, p.Name, p.X, p.Y)
	}
	b.WriteString("END\n")
	msg := b.String()
	// Send to all subs, remove closed
	for conn := range c.subs {
		conn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
		if _, err := conn.Write([]byte(msg)); err != nil {
			// drop
			delete(c.subs, conn)
			conn.Close()
		}
	}
}

package main

import (
	"bufio"
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Initialize UI and run the local game (independent process)
	interfaceIniciar()
	defer interfaceFinalizar()

	// Cria novo jogo
	jogo := jogoNovo()
	_ = jogoCarregarMapa("mapa.txt", &jogo) // mapa local inicial

	// Inicia sincronização com estado do client local (se disponível)
	addr := os.Getenv("GAME_STATE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:4001"
	}
	go startStateSync(&jogo, addr)

	// Inicia elementos concorrentes (monstro, etc.)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Mutex de mapa (para elementos que pedem acesso exclusivo)
	go func() {
		for {
			select {
			case ch := <-jogo.MapMutex:
				ch <- true
			case <-ctx.Done():
				return
			}
		}
	}()

	if jogo.Monstro != nil {
		go jogo.Monstro.Run(ctx, jogo.GameEvents, jogo.PlayerAlerts, jogo.PlayerState)
	}

	// Loop principal do jogo (não-bloqueante para processar eventos)
	evCh := interfaceLerEventoTecladoAsync()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ev := <-evCh:
			if ev.Tipo == "sair" {
				return
			}
			_ = personagemExecutarAcaoComCanal(ev, &jogo, jogo.PlayerState)
		case <-ticker.C:
			// processa eventos do jogo e redesenha periodicamente
			jogoProcessarEventos(&jogo)
			interfaceDesenharJogo(&jogo)
		}
	}
}

// Conecta ao broadcaster local do client (127.0.0.1:4001) e atualiza mapa/jogadores
func startStateSync(j *Jogo, addr string) {
	appliedMap := false
	for {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		rd := bufio.NewScanner(conn)
		rd.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var mapLines []string
		for rd.Scan() {
			line := strings.TrimSpace(rd.Text())
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "SELF ") {
				j.SelfID = strings.TrimSpace(strings.TrimPrefix(line, "SELF "))
			} else if strings.HasPrefix(line, "MAP ") {
				// read N lines
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
						mapLines = make([]string, 0, n)
						for i := 0; i < n && rd.Scan(); i++ {
							mapLines = append(mapLines, rd.Text())
						}
						// aplica mapa apenas uma vez ao conectar
						if len(mapLines) > 0 && !appliedMap {
							_ = jogoCarregarMapaDeLinhas(mapLines, j)
							appliedMap = true
						}
					}
				}
			} else if strings.HasPrefix(line, "PLAYERS ") {
				parts := strings.Fields(line)
				count := 0
				if len(parts) >= 2 {
					if n, err := strconv.Atoi(parts[1]); err == nil {
						count = n
					}
				}
				// zera players anteriores, vamos repovoar
				j.RemotePlayers = make(map[string]RemotePlayer, count)
				for i := 0; i < count && rd.Scan(); i++ {
					pl := strings.Split(rd.Text(), "\t")
					if len(pl) < 4 {
						continue
					}
					x, _ := strconv.Atoi(pl[2])
					y, _ := strconv.Atoi(pl[3])
					j.RemotePlayers[pl[0]] = RemotePlayer{ID: pl[0], Name: pl[1], X: x, Y: y}
				}
			} else if line == "END" {
				// snapshot completo recebido
			}
		}
		conn.Close()
		// reconectar em caso de fechamento
		time.Sleep(300 * time.Millisecond)
	}
}

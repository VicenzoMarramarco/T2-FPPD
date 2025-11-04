package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

// Elemento representa qualquer objeto do mapa (parede, personagem, vegetação, etc)
type Elemento struct {
	simbolo  rune
	cor      Cor
	corFundo Cor
	tangivel bool
}

type Jogo struct {
	Mapa              [][]Elemento // grade 2D representando o mapa
	PosX, PosY        int          // posição atual do personagem
	UltimoVisitado    Elemento
	StatusMsg         string
	InvisibleSteps    int
	DoubleJumps       int
	Monstro           *Monster
	InvisibilityItems []*Invisibility
	Stars             []*Star
	GameEvents        chan GameEvent
	PlayerState       chan PlayerState
	PlayerAlerts      chan PlayerAlert
	PlayerCollects    chan PlayerCollect
	StarCommands      chan StarCommand
	MapMutex          chan chan bool
	RemotePlayers     map[string]RemotePlayer // outros jogadores
	SelfID            string                  // id do jogador local (para não duplicar)
}

// Elementos visuais do jogo
var (
	Personagem           = Elemento{'☺', CorCinzaEscuro, CorPadrao, true}
	Inimigo              = Elemento{'☠', CorVermelho, CorPadrao, true}
	Parede               = Elemento{'▤', CorParede, CorFundoParede, true}
	Vegetacao            = Elemento{'♣', CorVerde, CorPadrao, false}
	Vazio                = Elemento{' ', CorPadrao, CorPadrao, false}
	InvisibilityItem     = Elemento{'¤', CorAmarelo, CorPadrao, false}
	PersonagemInvisivel  = Elemento{'☺', CorTexto, CorPadrao, true}
	StarElementVisible   = Elemento{'★', CorAmarelo, CorPadrao, false}
	StarElementInvisible = Elemento{' ', CorPadrao, CorPadrao, false}
	StarElementPulsing   = Elemento{'✦', CorCinzaEscuro, CorPadrao, false}
	StarElementCharging  = Elemento{'◉', CorVermelho, CorPadrao, false}
)

// Canal global para integração com o client.go
var PosUpdateChan chan [2]int

func jogoNovo() Jogo {
	return Jogo{
		UltimoVisitado: Vazio,
		GameEvents:     make(chan GameEvent, 10),
		PlayerState:    make(chan PlayerState, 10),
		PlayerAlerts:   make(chan PlayerAlert, 10),
		PlayerCollects: make(chan PlayerCollect, 10),
		StarCommands:   make(chan StarCommand, 10),
		MapMutex:       make(chan chan bool, 1),
		RemotePlayers:  make(map[string]RemotePlayer),
	}
}

// Lê um arquivo texto linha por linha e constrói o mapa do jogo
func jogoCarregarMapa(nome string, jogo *Jogo) error {
	arq, err := os.Open(nome)
	if err != nil {
		return err
	}
	defer arq.Close()

	scanner := bufio.NewScanner(arq)
	y := 0
	for scanner.Scan() {
		linha := scanner.Text()
		var linhaElems []Elemento
		for x, ch := range linha {
			e := Vazio
			switch ch {
			case Parede.simbolo:
				e = Parede
			case Inimigo.simbolo:
				e = Vazio
				if jogo.Monstro == nil {
					jogo.Monstro = &Monster{
						current_position: Position{X: x, Y: y},
						state:            Patrolling,
						destiny_position: Position{X: x + 5, Y: y + 5},
						id:               "monster_1",
					}
				}
			case Vegetacao.simbolo:
				e = Vegetacao
			case InvisibilityItem.simbolo:
				e = InvisibilityItem
				invisItem := &Invisibility{X: x, Y: y}
				jogo.InvisibilityItems = append(jogo.InvisibilityItems, invisItem)
			case '★':
				e = StarElementVisible
			case Personagem.simbolo:
				jogo.PosX, jogo.PosY = x, y
			}
			linhaElems = append(linhaElems, e)
		}
		jogo.Mapa = append(jogo.Mapa, linhaElems)
		y++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// Constrói o mapa a partir de linhas de texto fornecidas pelo servidor
func jogoCarregarMapaDeLinhas(linhas []string, jogo *Jogo) error {
	jogo.Mapa = nil
	jogo.Monstro = nil
	jogo.InvisibilityItems = nil
	y := 0
	for _, linha := range linhas {
		var linhaElems []Elemento
		for x, ch := range linha {
			e := Vazio
			switch ch {
			case Parede.simbolo:
				e = Parede
			case Inimigo.simbolo:
				e = Vazio
				if jogo.Monstro == nil {
					jogo.Monstro = &Monster{
						current_position: Position{X: x, Y: y},
						state:            Patrolling,
						destiny_position: Position{X: x + 5, Y: y + 5},
						id:               "monster_1",
					}
				}
			case Vegetacao.simbolo:
				e = Vegetacao
			case InvisibilityItem.simbolo:
				e = InvisibilityItem
				invisItem := &Invisibility{X: x, Y: y}
				jogo.InvisibilityItems = append(jogo.InvisibilityItems, invisItem)
			case '★':
				e = StarElementVisible
			case Personagem.simbolo:
				// Quando carregando mapa recebido do servidor, não reposiciona o jogador local.
				// Trata como espaço vazio para evitar "teleporte" para a posição inicial.
				e = Vazio
			}
			linhaElems = append(linhaElems, e)
		}
		jogo.Mapa = append(jogo.Mapa, linhaElems)
		y++
	}
	return nil
}

// Verifica se o personagem pode se mover para a posição (x, y)
func jogoPodeMoverPara(jogo *Jogo, x, y int) bool {
	if y < 0 || y >= len(jogo.Mapa) {
		return false
	}
	if x < 0 || x >= len(jogo.Mapa[y]) {
		return false
	}
	if jogo.Mapa[y][x].tangivel {
		return false
	}
	return true
}

// Move um elemento e notifica cliente e servidor
func jogoMoverElemento(jogo *Jogo, x, y, dx, dy int) {
	nx, ny := x+dx, y+dy
	elemento := jogo.Mapa[y][x]

	jogo.Mapa[y][x] = jogo.UltimoVisitado
	jogo.UltimoVisitado = jogo.Mapa[ny][nx]
	jogo.Mapa[ny][nx] = elemento
	jogo.PosX, jogo.PosY = nx, ny

	// Envia posição para o cliente local e canal global
	jogoEnviarEstadoJogador(jogo)
	jogo.ReportarMovimento()
}

func (j *Jogo) elementoJogador() Elemento {
	if j.InvisibleSteps > 0 {
		return PersonagemInvisivel
	}
	return Personagem
}

func jogoProcessarEventos(jogo *Jogo) {
	select {
	case event := <-jogo.GameEvents:
		jogoTratarEvento(jogo, event)
	default:
	}
}

func jogoTratarEvento(jogo *Jogo, event GameEvent) {
	switch event.Type {
	case "monster_move":
		if data, ok := event.Data.(MonsterMoveData); ok {
			if jogoPodeMoverPara(jogo, data.NewX, data.NewY) {
				if jogo.Monstro != nil && jogo.Monstro.id == data.MonsterID {
					jogo.Monstro.current_position = Position{X: data.NewX, Y: data.NewY}
					if data.NewX == jogo.PosX && data.NewY == jogo.PosY {
						collisionEvent := GameEvent{
							Type: "monster_collision",
							Data: map[string]interface{}{"x": data.NewX, "y": data.NewY},
						}
						select {
						case jogo.GameEvents <- collisionEvent:
						default:
						}
					}
				}
			}
		}
	case "monster_collision":
		jogo.StatusMsg = "Pego pelo monstro!"
	case EventApplyInvisibility:
		if data, ok := event.Data.(InvisibilityApplied); ok {
			jogo.InvisibleSteps = data.Duration
			jogo.StatusMsg = "Invisibilidade coletada!"
		}
	case "ApplyDoubleJump":
		if data, ok := event.Data.(DoubleJumpApplied); ok {
			jogo.DoubleJumps = data.Jumps
			jogo.StatusMsg = "Estrela coletada! Pulo duplo ativado!"
		}
	}
}

// Notifica o client.go via TCP
func jogoEnviarEstadoJogador(jogo *Jogo) {
	go func(x, y int) {
		addr := os.Getenv("GAME_CMD_ADDR")
		if addr == "" {
			addr = "127.0.0.1:4000"
		}
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err != nil {
			return
		}
		defer conn.Close()
		// Envia comando
		fmt.Fprintf(conn, "MOVE %d %d\n", x, y)

		// Handshake: tenta ler uma resposta rápida antes de fechar (timeout curto)
		_ = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		reader := bufio.NewReader(conn)
		_, _ = reader.ReadString('\n')
	}(jogo.PosX, jogo.PosY)
}

// Envia posição pro canal PosUpdateChan (usado pelo client.go)
func (j *Jogo) ReportarMovimento() {
	if PosUpdateChan != nil {
		select {
		case PosUpdateChan <- [2]int{j.PosX, j.PosY}:
		default:
		}
	}
}

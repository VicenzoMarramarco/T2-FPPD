// element_star.go - Implementação do elemento estrela concorrente
package main

import (
	"context"
	"math/rand"
	"time"
)

// Estados da estrela
type StarState int

const (
	StarVisible StarState = iota
	StarInvisible
	StarPulsing
	StarCharging
)

// Tipos de eventos da estrela
const (
	EventStarCollected    = "StarCollected"
	EventStarStateChange  = "StarStateChange"
	EventStarPulse        = "StarPulse"
	EventStarCharged      = "StarCharged"
	EventStarTimeout      = "StarTimeout"
	EventStarCommunicate  = "StarCommunicate"
	EventRequestMapAccess = "RequestMapAccess"
	EventMapAccessGranted = "MapAccessGranted"
)

// Duração dos estados da estrela
const (
	StarVisibilityDuration = 8 * time.Second  // Tempo visível
	StarInvisibleDuration  = 4 * time.Second  // Tempo invisível
	StarPulseDuration      = 2 * time.Second  // Duração de uma pulsação
	StarChargeDuration     = 10 * time.Second // Tempo para carregar
	StarTimeoutDuration    = 15 * time.Second // Timeout para mudança de comportamento
)

type StarCollectedData struct {
	X, Y      int
	BonusType string //
	Value     int
}

type StarStateChangeData struct {
	X, Y     int
	OldState StarState
	NewState StarState
	StarID   string
}

type StarPulseData struct {
	X, Y       int
	IsVisible  bool
	PulseCount int
}

type StarChargedData struct {
	X, Y     int
	Energy   int
	Duration time.Duration
}

type StarTimeoutData struct {
	X, Y    int
	Message string
	Action  string
	StarID  string
}

type StarCommunicationData struct {
	FromStarID string
	ToStarID   string
	Message    string
	Data       interface{}
}

type StarCommand struct {
	Type   string
	Target string
	Data   interface{}
}

// Estrutura da estrela
type Star struct {
	X, Y          int
	State         StarState
	ID            string
	IsVisible     bool
	Energy        int
	PulseCount    int
	LastPlayerPos Position
	MapAccess     chan chan bool
}

// Cria uma nova estrela
func NewStar(x, y int, id string) *Star {
	return &Star{
		X:         x,
		Y:         y,
		State:     StarVisible,
		ID:        id,
		IsVisible: true,
		Energy:    0,
		MapAccess: make(chan chan bool, 1),
	}
}

func (s *Star) Run(ctx context.Context, gameEvents chan<- GameEvent, playerState <-chan PlayerState,
	playerCollects <-chan PlayerCollect, starCommands <-chan StarCommand, mapMutex chan chan bool) {

	// Timers para diferentes comportamentos
	visibilityTimer := time.NewTimer(StarVisibilityDuration)
	pulseTimer := time.NewTimer(StarPulseDuration)
	chargeTimer := time.NewTimer(StarChargeDuration)
	timeoutTimer := time.NewTimer(StarTimeoutDuration)

	defer visibilityTimer.Stop()
	defer pulseTimer.Stop()
	defer chargeTimer.Stop()
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case playerPos := <-playerState:
			s.LastPlayerPos = Position(playerPos)
			s.handlePlayerMovement(gameEvents, playerPos)

		case collect := <-playerCollects:
			if collect.X == s.X && collect.Y == s.Y && s.IsVisible && s.State == StarVisible {
				s.handleCollection(gameEvents, collect)
				return // Estrela coletada, termina goroutine
			}

		case command := <-starCommands:
			s.handleStarCommand(gameEvents, command)

		case <-timeoutTimer.C:
			s.handleTimeout(gameEvents)
			timeoutTimer.Reset(StarTimeoutDuration)

		case <-visibilityTimer.C:
			s.toggleVisibility(gameEvents, mapMutex)
			visibilityTimer.Reset(s.getNextVisibilityDuration())

		case <-pulseTimer.C:
			if s.State == StarPulsing {
				s.handlePulse(gameEvents, mapMutex)
				pulseTimer.Reset(StarPulseDuration)
			}

		case <-chargeTimer.C:
			if s.State == StarCharging {
				s.handleChargeComplete(gameEvents)
				chargeTimer.Reset(StarChargeDuration)
			}

		// REQUISITO: Exclusão mútua usando canais
		case responseChan := <-s.MapAccess:
			s.requestMapAccess(mapMutex, responseChan)
		}
	}
}

func (s *Star) handlePlayerMovement(gameEvents chan<- GameEvent, playerPos PlayerState) {
	distance := abs(playerPos.X-s.X) + abs(playerPos.Y-s.Y) // Distância Manhattan

	if distance <= 1 && s.State != StarPulsing {
		s.changeState(StarPulsing, gameEvents)
	} else if distance > 3 && s.State == StarPulsing {
		s.changeState(StarVisible, gameEvents)
	}
}

// Manipula coleta da estrela
func (s *Star) handleCollection(gameEvents chan<- GameEvent, collect PlayerCollect) {
	bonusType := "score"
	value := 100

	// Bônus especial se coletada em estado especial
	switch s.State {
	case StarPulsing:
		bonusType = "power"
		value = 300
	case StarCharging:
		bonusType = "life"
		value = 1
	default:
		value += s.Energy * 10
	}

	gameEvents <- GameEvent{
		Type: EventStarCollected,
		Data: StarCollectedData{
			X:         s.X,
			Y:         s.Y,
			BonusType: bonusType,
			Value:     value,
		},
	}

	// Remover estrela do mapa
	gameEvents <- GameEvent{
		Type: EventRemoveElement,
		Data: StarBonus{X: s.X, Y: s.Y},
	}
}

func (s *Star) handleStarCommand(gameEvents chan<- GameEvent, command StarCommand) {
	switch command.Type {
	case "change_state":
		if data, ok := command.Data.(StarState); ok {
			s.changeState(data, gameEvents)
		}
	case "pulse":
		if s.State != StarPulsing {
			s.changeState(StarPulsing, gameEvents)
		}
	case "charge":
		if s.State != StarCharging {
			s.changeState(StarCharging, gameEvents)
		}
	case "communicate":
		if data, ok := command.Data.(StarCommunicationData); ok {
			s.handleCommunication(gameEvents, data)
		}
	}
}

func (s *Star) handleTimeout(gameEvents chan<- GameEvent) {
	// Comportamento alternativo quando não recebe interação por tempo limite
	actions := []string{"charge", "pulse", "hide", "energy_burst"}
	action := actions[rand.Intn(len(actions))]

	switch action {
	case "charge":
		s.changeState(StarCharging, gameEvents)
	case "pulse":
		s.changeState(StarPulsing, gameEvents)
	case "hide":
		s.changeState(StarInvisible, gameEvents)
	case "energy_burst":
		s.Energy += 50
		gameEvents <- GameEvent{
			Type: EventStarCharged,
			Data: StarChargedData{
				X:        s.X,
				Y:        s.Y,
				Energy:   s.Energy,
				Duration: StarChargeDuration,
			},
		}
	}

	gameEvents <- GameEvent{
		Type: EventStarTimeout,
		Data: StarTimeoutData{
			X:       s.X,
			Y:       s.Y,
			Message: "Estrela mudou comportamento por timeout",
			Action:  action,
			StarID:  s.ID,
		},
	}
}

// Alterna visibilidade da estrela
func (s *Star) toggleVisibility(gameEvents chan<- GameEvent, mapMutex chan chan bool) {
	// REQUISITO: Exclusão mútua para acesso ao mapa
	responseChan := make(chan bool)
	mapMutex <- responseChan
	<-responseChan // Aguarda liberação do acesso

	s.IsVisible = !s.IsVisible

	if s.IsVisible {
		s.changeState(StarVisible, gameEvents)
	} else {
		s.changeState(StarInvisible, gameEvents)
	}
}

// Manipula pulsação da estrela
func (s *Star) handlePulse(gameEvents chan<- GameEvent, mapMutex chan chan bool) {
	responseChan := make(chan bool)
	mapMutex <- responseChan
	<-responseChan // Exclusão mútua

	s.IsVisible = !s.IsVisible
	s.PulseCount++

	gameEvents <- GameEvent{
		Type: EventStarPulse,
		Data: StarPulseData{
			X:          s.X,
			Y:          s.Y,
			IsVisible:  s.IsVisible,
			PulseCount: s.PulseCount,
		},
	}

	if s.PulseCount >= 10 {
		s.PulseCount = 0
		s.changeState(StarVisible, gameEvents)
	}
}

// Manipula carregamento completo de energia
func (s *Star) handleChargeComplete(gameEvents chan<- GameEvent) {
	s.Energy += 100

	gameEvents <- GameEvent{
		Type: EventStarCharged,
		Data: StarChargedData{
			X:        s.X,
			Y:        s.Y,
			Energy:   s.Energy,
			Duration: StarChargeDuration,
		},
	}

	s.changeState(StarVisible, gameEvents)
}

// Manipula comunicação entre estrelas
func (s *Star) handleCommunication(gameEvents chan<- GameEvent, data StarCommunicationData) {
	switch data.Message {
	case "sync_pulse":
		s.changeState(StarPulsing, gameEvents)
	case "share_energy":
		if energy, ok := data.Data.(int); ok {
			s.Energy += energy / 2
		}
	case "warning":
		s.changeState(StarCharging, gameEvents)
	}

	gameEvents <- GameEvent{
		Type: EventStarCommunicate,
		Data: data,
	}
}

// Muda estado da estrela
func (s *Star) changeState(newState StarState, gameEvents chan<- GameEvent) {
	oldState := s.State
	s.State = newState

	switch newState {
	case StarVisible:
		s.IsVisible = true
	case StarInvisible:
		s.IsVisible = false
	case StarPulsing:
	case StarCharging:
		s.IsVisible = true
	}

	gameEvents <- GameEvent{
		Type: EventStarStateChange,
		Data: StarStateChangeData{
			X:        s.X,
			Y:        s.Y,
			OldState: oldState,
			NewState: newState,
			StarID:   s.ID,
		},
	}
}

// Exclusão mútua
func (s *Star) requestMapAccess(mapMutex chan chan bool, responseChan chan bool) {
	mapMutex <- responseChan
}

func (s *Star) getNextVisibilityDuration() time.Duration {
	if s.IsVisible {
		return StarVisibilityDuration
	}
	return StarInvisibleDuration
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (s *StarBonus) Run(ctx context.Context, out chan<- GameEvent, collected <-chan PlayerCollect) {
	// Converte StarBonus para Star para usar a implementação completa
	star := NewStar(s.X, s.Y, "legacy_star")

	playerState := make(chan PlayerState, 10)
	starCommands := make(chan StarCommand, 10)
	mapMutex := make(chan chan bool, 1)

	// Goroutine para gerenciar exclusão mútua
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case responseChan := <-mapMutex:
				responseChan <- true
			}
		}
	}()

	star.Run(ctx, out, playerState, collected, starCommands, mapMutex)
}



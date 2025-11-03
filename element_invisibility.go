package main

import "context"

const InvisibilityDuration = 20

// Eventos produzidos pelo elemento
const (
	EventApplyInvisibility = "ApplyInvisibility"
	EventRemoveElement     = "RemoveElement"
)

// Payload para aplicação de invisibilidade
type InvisibilityApplied struct {
	Duration int
}

func (i *Invisibility) Run(ctx context.Context, out chan<- GameEvent, picked <-chan PlayerCollect) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-picked:
			if !ok {
				return
			}
			if ev.X != i.X || ev.Y != i.Y {
				continue
			}

			out <- GameEvent{
				Type: EventRemoveElement,
				Data: Invisibility{X: i.X, Y: i.Y},
			}

			// Aplica o buff de invisibilidade ao jogador
			out <- GameEvent{
				Type: EventApplyInvisibility,
				Data: InvisibilityApplied{Duration: InvisibilityDuration},
			}

			// Item é one-shot
			return
		}
	}
}

// Remoção do item “sob” o jogador quando consumido
func ConsumirItemInvisibilidade(jogo *Jogo) bool {
	if jogo.UltimoVisitado.simbolo == InvisibilityItem.simbolo {
		jogo.UltimoVisitado = Vazio
		return true
	}
	return false
}

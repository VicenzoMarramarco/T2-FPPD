// personagem.go - Funções para movimentação e ações do personagem
package main

import (
	"fmt"
	"math/rand"
)

// Atualiza a posição do personagem com base na tecla pressionada (WASD)
func personagemMover(tecla rune, jogo *Jogo) {
	dx, dy := 0, 0
	switch tecla {
	case 'w':
		dy = -1 // Move para cima
	case 'a':
		dx = -1 // Move para a esquerda
	case 's':
		dy = 1 // Move para baixo
	case 'd':
		dx = 1 // Move para a direita
	}

	// Verificar se tem pulos duplos disponíveis
	stepSize := 1
	if jogo.DoubleJumps > 0 {
		stepSize = 2
		jogo.StatusMsg = fmt.Sprintf("Pulo duplo! Restam %d pulos", jogo.DoubleJumps-1)
	}

	nx, ny := jogo.PosX+(dx*stepSize), jogo.PosY+(dy*stepSize)

	if jogoPodeMoverPara(jogo, nx, ny) {
		if stepSize == 2 {
			intermediateX, intermediateY := jogo.PosX+dx, jogo.PosY+dy
			if !jogoPodeMoverPara(jogo, intermediateX, intermediateY) {
				nx, ny = jogo.PosX+dx, jogo.PosY+dy
				stepSize = 1
				if !jogoPodeMoverPara(jogo, nx, ny) {
					return
				}
				jogo.StatusMsg = fmt.Sprintf("Pulo duplo bloqueado! Restam %d pulos", jogo.DoubleJumps)
			} else {
				jogo.DoubleJumps--
				if jogo.DoubleJumps == 0 {
					jogo.StatusMsg = "Último pulo duplo usado!"
				}
			}
		}

		jogoMoverElemento(jogo, jogo.PosX, jogo.PosY, dx*stepSize, dy*stepSize)
		jogo.PosX, jogo.PosY = nx, ny

		// Verificar se coletou estrela
		coletouEstrela := ConsumirItemEstrela(jogo)
		if coletouEstrela {
			jogo.DoubleJumps = 3
			jogo.StatusMsg = "Estrela coletada! 3 pulos duplos concedidos!"
		}

		// Verificar se coletou item de invisibilidade
		coletouInvisibilidade := ConsumirItemInvisibilidade(jogo)
		if coletouInvisibilidade {
			jogo.InvisibleSteps = InvisibilityDuration
			jogo.StatusMsg = "Invisibilidade coletada!"
		}

		// Notifica o canal de coleta
		collectEvent := PlayerCollect{
			X: jogo.PosX,
			Y: jogo.PosY,
		}
		select {
		case jogo.PlayerCollects <- collectEvent:
		default:
		}

		// Atualiza invisibilidade restante
		if !coletouInvisibilidade && !coletouEstrela && jogo.InvisibleSteps > 0 {
			jogo.InvisibleSteps--
			if jogo.InvisibleSteps == 0 {
				jogo.StatusMsg = "Invisibilidade expirou"
			} else {
				jogo.StatusMsg = fmt.Sprintf("Invisível: %d movimentos restantes", jogo.InvisibleSteps)
			}
		}
	} else {
		if stepSize == 2 {
			nx, ny = jogo.PosX+dx, jogo.PosY+dy
			if jogoPodeMoverPara(jogo, nx, ny) {
				jogoMoverElemento(jogo, jogo.PosX, jogo.PosY, dx, dy)
				jogo.PosX, jogo.PosY = nx, ny
				jogo.StatusMsg = fmt.Sprintf("Pulo duplo bloqueado - movimento normal. Restam %d pulos", jogo.DoubleJumps)

				coletouEstrela := ConsumirItemEstrela(jogo)
				if coletouEstrela {
					jogo.DoubleJumps = 3
					jogo.StatusMsg = "Estrela coletada! 3 pulos duplos concedidos!"
				}

				coletouInvisibilidade := ConsumirItemInvisibilidade(jogo)
				if coletouInvisibilidade {
					jogo.InvisibleSteps = InvisibilityDuration
					jogo.StatusMsg = "Invisibilidade coletada!"
				}
			}
		}
	}
}

func personagemInteragir(jogo *Jogo) {
	jogo.StatusMsg = fmt.Sprintf("Interagindo em (%d, %d)", jogo.PosX, jogo.PosY)
}
// Envia um alerta simples do jogador (ex: barulho, aviso, etc)
func jogoEnviarAlerta(jogo *Jogo, tipoAlerta string) {
	alert := PlayerAlert{
		Type: tipoAlerta,
		Data: map[string]int{
			"x": jogo.PosX,
			"y": jogo.PosY,
		},
	}

	select {
	case jogo.PlayerAlerts <- alert:
	default:
	}
}


// Versão chamada quando o jogo principal trata eventos de teclado simples
func personagemExecutarAcao(ev EventoTeclado, jogo *Jogo) bool {
	switch ev.Tipo {
	case "sair":
		return false

	case "interagir":
		personagemInteragir(jogo)

	case "mover":
		personagemMover(ev.Tecla, jogo)

		// Envia posição para o cliente local (127.0.0.1:4000)
		jogoEnviarEstadoJogador(jogo)

		// Envia evento de barulho para o monstro (20% de chance)
		if rand.Float32() < 0.2 {
			jogoEnviarAlerta(jogo, "noise")
		}
	}
	return true
}

// Versão com canal (usada no loop principal do jogo)
func personagemExecutarAcaoComCanal(ev EventoTeclado, jogo *Jogo, playerChannel chan<- PlayerState) bool {
	switch ev.Tipo {
	case "sair":
		return false

	case "interagir":
		personagemInteragir(jogo)

	case "mover":
		personagemMover(ev.Tecla, jogo)

		// Envia estado pro canal interno (mantém compatibilidade)
		playerState := PlayerState{
			X: jogo.PosX,
			Y: jogo.PosY,
		}
		select {
		case playerChannel <- playerState:
		default:
		}

		// *** ADIÇÃO: notificar o cliente local via TCP ***
		jogoEnviarEstadoJogador(jogo)

		// *** OPCIONAL: chance de barulho ***
		if rand.Float32() < 0.2 {
			jogoEnviarAlerta(jogo, "noise")
		}
	}
	return true
}


// Remoção da estrela "sob" o jogador quando consumida.
func ConsumirItemEstrela(jogo *Jogo) bool {
	if jogo.UltimoVisitado.simbolo == StarElementVisible.simbolo {
		jogo.UltimoVisitado = Vazio
		return true
	}
	return false
}

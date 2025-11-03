package main

import (
	"github.com/nsf/termbox-go"
)

// Declaração da função existente em outro arquivo


// Define um tipo Cor para encapsuladar as cores do termbox
type Cor = termbox.Attribute

// Definições de cores utilizadas no jogo
const (
	CorPadrao      Cor = termbox.ColorDefault
	CorCinzaEscuro     = termbox.ColorDarkGray
	CorVermelho        = termbox.ColorRed
	CorVerde           = termbox.ColorGreen
	CorParede          = termbox.ColorBlack | termbox.AttrBold | termbox.AttrDim
	CorFundoParede     = termbox.ColorDarkGray
	CorTexto           = termbox.ColorDarkGray
	CorAmarelo         = termbox.ColorYellow
)

// EventoTeclado representa uma ação detectada do teclado (como mover, sair ou interagir)
type EventoTeclado struct {
	Tipo  string 
	Tecla rune 
}

// Inicializa a interface gráfica usando termbox
func interfaceIniciar() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
}

// Encerra o uso da interface termbox
func interfaceFinalizar() {
	termbox.Close()
}




// Lê um evento do teclado e o traduz para um EventoTeclado
func interfaceLerEventoTeclado() EventoTeclado {
	ev := termbox.PollEvent()
	if ev.Type != termbox.EventKey {
		return EventoTeclado{}
	}
	if ev.Key == termbox.KeyEsc {
		return EventoTeclado{Tipo: "sair"}
	}
	if ev.Ch == 'e' {
		return EventoTeclado{Tipo: "interagir"}
	}
	return EventoTeclado{Tipo: "mover", Tecla: ev.Ch}
}

// Renderiza todo o estado atual do jogo na tela
func interfaceDesenharJogo(jogo *Jogo) {
	interfaceLimparTela()

	// Desenha todos os elementos do mapa
	for y, linha := range jogo.Mapa {
		for x, elem := range linha {
			interfaceDesenharElemento(x, y, elem)
		}
	}

	// Desenha o personagem sobre o mapa
	interfaceDesenharElemento(jogo.PosX, jogo.PosY, jogo.elementoJogador())

	// Desenha o monstro se existir
	if jogo.Monstro != nil {
		interfaceDesenharElemento(jogo.Monstro.current_position.X, jogo.Monstro.current_position.Y, Inimigo)
	}

	
// Desenha as estrelas
	for _, star := range jogo.Stars {
		if star.IsVisible {
			starElement := jogoGetStarElement(star)
			interfaceDesenharElemento(star.X, star.Y, starElement)
		}
	}
	interfaceDesenharBarraDeStatus(jogo)
	interfaceAtualizarTela()
}

func interfaceLimparTela() {
	termbox.Clear(CorPadrao, CorPadrao)
}

func interfaceAtualizarTela() {
	termbox.Flush()
}

func interfaceDesenharElemento(x, y int, elem Elemento) {
	termbox.SetCell(x, y, elem.simbolo, elem.cor, elem.corFundo)
}

func interfaceDesenharBarraDeStatus(jogo *Jogo) {

	for i, c := range jogo.StatusMsg {
		termbox.SetCell(i, len(jogo.Mapa)+1, c, CorTexto, CorPadrao)
	}

	// Instruções fixas
	msg := "Use WASD para mover e E para interagir. ESC para sair."
	for i, c := range msg {
		termbox.SetCell(i, len(jogo.Mapa)+3, c, CorTexto, CorPadrao)
	}
}

// Versão assíncrona de leitura de eventos do teclado (não-bloqueante)
func interfaceLerEventoTecladoAsync() <-chan EventoTeclado {
	ch := make(chan EventoTeclado, 1)
	go func() {
		defer close(ch)
		for {
			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				evento := EventoTeclado{}
				switch ev.Key {
				case termbox.KeyEsc:
					evento.Tipo = "sair"
					ch <- evento
					return
				default:
					switch ev.Ch {
					case 'e', 'E':
						evento.Tipo = "interagir"
					case 'w', 'W', 'a', 'A', 's', 'S', 'd', 'D':
						evento.Tipo = "mover"
						evento.Tecla = ev.Ch
					default:
						continue // Ignorar outras teclas
					}
				}
				ch <- evento
			}
		}
	}()
	return ch
	
}



// jogoGetStarElement returns visual Elemento for a Star based on its state.
func jogoGetStarElement(star *Star) Elemento {
    if star == nil {
        return Vazio
    }
    switch star.State {
    case StarVisible:
        return StarElementVisible
    case StarInvisible:
        return StarElementInvisible
    case StarPulsing:
        return StarElementPulsing
    case StarCharging:
        return StarElementCharging
    default:
        return StarElementVisible
    }
}

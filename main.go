package main

import (
	"context"
	"time"
)

func main() {
	// Initialize UI and run the local game (independent process)
	interfaceIniciar()
	defer interfaceFinalizar()

	// Cria novo jogo
	jogo := jogoNovo()
	_ = jogoCarregarMapa("mapa.txt", &jogo) // certifique-se de ter o arquivo mapa.txt

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

# Jogo de Terminal em Go

Este projeto é um pequeno jogo desenvolvido em Go que roda no terminal usando a biblioteca [termbox-go](https://github.com/nsf/termbox-go). O jogador controla um personagem que pode se mover por um mapa carregado de um arquivo de texto.

## Como funciona

- O mapa é carregado de um arquivo `.txt` contendo caracteres que representam diferentes elementos do jogo.
- O personagem se move com as teclas **W**, **A**, **S**, **D**.
- Pressione **E** para interagir com o ambiente.
- Pressione **ESC** para sair do jogo.

### Controles

| Tecla | Ação              |
|-------|-------------------|
| W     | Mover para cima   |
| A     | Mover para esquerda |
| S     | Mover para baixo  |
| D     | Mover para direita |
| E     | Interagir         |
| ESC   | Sair do jogo      |

## Como compilar

1. Instale o Go e clone este repositório.
2. Navegue para a pasta do projeto:

```bash
cd T2-FPPD
```

3. Inicialize um novo módulo Go e instale dependências:

```bash
go mod init jogo
go get -u github.com/nsf/termbox-go
```

4. Compile o programa:

Linux:

```bash
go build -o jogo
```

Windows:

```bash
go build -o jogo.exe
```

Também é possivel compilar o projeto usando o comando `make` no Linux ou o script `build.bat` no Windows.

## Como executar

Execução em terminais separados (recomendado):

Terminal 1 – Servidor RPC
```powershell
go run ./cmd/server
```

Terminal 2 – Cliente RPC + listener local (recebe "MOVE x y" do jogo)
```powershell
go run ./cmd/client --name "Player1" --addr "10.135.177.130:12345" --ui "127.0.0.1:4001" --listen "127.0.0.1:4000"
```

Terminal 3 – Jogo (termbox)
```powershell
$env:GAME_STATE_ADDR = "127.0.0.1:4001"
$env:GAME_CMD_ADDR = "127.0.0.1:4000"
go run .
```

Terminal 4 – Cliente RPC + listener local (recebe "MOVE x y" do jogo)
```powershell
go run ./cmd/client --name "Player2" --addr "10.135.177.130:12345" --ui "127.0.0.1:4002" --listen "127.0.0.1:4003"
```

Terminal 5 – Jogo (termbox)
```powershell
$env:GAME_STATE_ADDR = "127.0.0.1:4002"
$env:GAME_CMD_ADDR = "127.0.0.1:4003"
go run .
```
##
By: Vicenzo Martins Marramarco
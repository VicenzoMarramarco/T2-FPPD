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
cd fppd-jogo
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
go run ./cmd/client --name "Player1" --addr "10.132.248.167:12345" --ui "127.0.0.1:4001" --listen "127.0.0.1:4000"```


Terminal 3 – Jogo (termbox)
```powershell
$env:GAME_STATE_ADDR = "127.0.0.1:4001"
$env:GAME_CMD_ADDR = "127.0.0.1:4000"
go run .
```

Terminal 4 – Cliente RPC + listener local (recebe "MOVE x y" do jogo)
```powershell
go run ./cmd/client --name "Player2" --addr "10.132.248.167:12345" --ui "127.0.0.1:4002" --listen "127.0.0.1:4003"


Terminal 5 – Jogo (termbox)
```powershell
$env:GAME_STATE_ADDR = "127.0.0.1:4002"
$env:GAME_CMD_ADDR = "127.0.0.1:4003"
go run .
```



O jogo abre no terminal. Use WASD para mover, E para interagir e ESC para sair. A cada movimento o jogo envia "MOVE x y" para 127.0.0.1:4000; o cliente repassa ao servidor via RPC.

### Modo Multijogador (mapa compartilhado e visualização dos jogadores)

Agora o servidor fornece o mesmo mapa para todos os clientes e cada cliente transmite o estado (mapa e jogadores) localmente para o jogo (UI). Isso permite abrir dois clientes e ver ambos no mesmo mapa.

Execute em terminais separados por cliente:

1) Servidor RPC (uma vez só)
```powershell
go run ./cmd/server
```

2) Cliente 1 (com porta de UI 4001)
```powershell
go run ./cmd/client --name "Player1" --addr "localhost:12345" --ui "127.0.0.1:4001"
```

3) Jogo 1 (apontando para a porta de UI do Cliente 1)
```powershell
$env:GAME_STATE_ADDR = "127.0.0.1:4001"
go run .
```

4) Cliente 2 (com outra porta de UI, por exemplo 4002)
```powershell
go run ./cmd/client --name "Player2" --addr "localhost:12345" --ui "127.0.0.1:4002"
```

5) Jogo 2 (apontando para a porta de UI do Cliente 2)
```powershell
$env:GAME_STATE_ADDR = "127.0.0.1:4002"
go run .
```

Notas:
- Cada instância do jogo (UI) renderiza: o jogador local (cinza) e os demais jogadores (ciano) vindos do estado do cliente.
- O mapa é carregado do servidor e substitui o mapa local automaticamente quando recebido.
- Se não definir `GAME_STATE_ADDR`, a UI tenta `127.0.0.1:4001` por padrão.

## Estrutura do projeto

- main.go — Ponto de entrada e loop principal
- interface.go — Entrada, saída e renderização com termbox
- jogo.go — Estruturas e lógica do estado do jogo
- personagem.go — Ações do jogador

Observações:
- Para alterar portas, use `--addr` ao iniciar o servidor e cliente.
- O cliente precisa estar rodando para o jogo conseguir enviar os comandos via TCP (porta 4000).
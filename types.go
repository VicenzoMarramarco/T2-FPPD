// types.go - Definições de tipos para elementos especiais
package main

type Position struct {
	X, Y int
}

// Estados do monstro
type MonsterState int

const (
	Hunting    MonsterState = iota 
	Patrolling                 
)

// Structs dos elementos especiais
type Monster struct {
	current_position Position     // Posição atual do monster
	shift_count      int          // Contador para movimento a cada 2 turnos
	destiny_position Position     // Posição de destino (patrulha)
	last_seen        Position     // Última posição vista do jogador
	state            MonsterState // Estado atual (hunting/patrolling)
	id               string       // ID único do monster
}

type StarBonus struct {
	X, Y int // Posição da estrela
}

type Invisibility struct {
	X, Y int // Posição do item de invisibilidade
}

type GameEvent struct {
	Type string      
	Data interface{}
}

type MonsterMoveData struct {
	OldX, OldY int    
	NewX, NewY int    
	MonsterID  string 
}

type PlayerAlert struct {
	Type string     
	Data interface{} 
}

type PlayerState struct {
	X, Y int 
}

type PlayerCollect struct {
	X, Y int // Posição onde o jogador coletou algo
}

type DoubleJumpApplied struct {
	Jumps int // Número de pulos duplos concedidos
}

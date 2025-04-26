package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand/v2"
	"sync/atomic"

	"github.com/godbus/dbus"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
	paddleWidth  = 10
	paddleHeight = 80
	ballRadius   = 5.0
	ballAccel    = 1.2
	player2Speed = 1.8
)

var (
	// volume stores the latest AirPods volume (0-127)
	volume uint32 = 63
)

type Ball struct {
	x      float32
	y      float32
	speedX float32
	speedY float32
}

type Player struct {
	points int
	x      float32
	y      float32
}

// Game holds the game state
type Game struct {
	ball    Ball
	player1 Player
	player2 Player
}

// Update is called every tick (1/60 [s] by default)
func (g *Game) Update() error {
	// nothing to update here; paddle is drawn based on volume

	// move ball
	g.ball.x += g.ball.speedX
	g.ball.y += g.ball.speedY

	// check for collisions with player1 paddle
	if g.ball.x-ballRadius <= g.player1.x+paddleWidth && g.ball.y >= g.player1.y && g.ball.y <= g.player1.y+paddleHeight {
		g.ball.speedX = -(g.ball.speedX * ballAccel)
		g.ball.speedY = (g.ball.speedY * ballAccel)
	}

	// check for collisions with player2 paddle
	if g.ball.x+ballRadius >= g.player2.x && g.ball.y >= g.player2.y && g.ball.y <= g.player2.y+paddleHeight {
		g.ball.speedX = -(g.ball.speedX * ballAccel)
		g.ball.speedY = (g.ball.speedY * ballAccel)
	}

	// check for collisions with top and bottom of screen
	if g.ball.y-ballRadius <= 0 || g.ball.y+ballRadius >= screenHeight {
		g.ball.speedY = -g.ball.speedY
	}

	// move player2 paddle
	if g.ball.y > g.player2.y+paddleHeight/2 && g.ball.x > screenWidth/2 {
		g.player2.y += player2Speed
	} else if g.ball.y < g.player2.y+paddleHeight/2 && g.ball.x > screenWidth/2 {
		g.player2.y -= player2Speed
	}

	// check for out of bounds
	if g.ball.x-ballRadius <= 0 {
		g.player2.points++
		g.ball.x = screenWidth / 2
		g.ball.y = screenHeight / 2
		g.ball.speedX = -2
		g.ball.speedY = -1 + rand.Float32()*2
	}
	if g.ball.x+ballRadius >= screenWidth {
		g.player1.points++
		g.ball.x = screenWidth / 2
		g.ball.y = screenHeight / 2
		g.ball.speedX = 2
		g.ball.speedY = 1 + rand.Float32()*2
	}

	return nil
}

// Draw draws the game screen
func (g *Game) Draw(screen *ebiten.Image) {
	// clear screen
	screen.Fill(color.Black)

	// map volume (0..127) to paddle Y (0..max)
	vol := atomic.LoadUint32(&volume)
	g.player1.y = float32(screenHeight-paddleHeight) * (float32(vol) / 127.0)

	// draw ball as a filled circle
	vector.DrawFilledCircle(screen, g.ball.x, g.ball.y, ballRadius, color.White, false)

	// draw player1 paddle as a filled rectangle
	vector.DrawFilledRect(screen, g.player1.x, g.player1.y, paddleWidth, paddleHeight, color.White, false)

	// draw player2 paddle
	vector.DrawFilledRect(screen, g.player2.x, g.player2.y, paddleWidth, paddleHeight, color.White, false)

	// debug print
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Volume: %d", vol), 10, 10)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Player 1: %d", g.player1.points), 10, 30)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Player 2: %d", g.player2.points), 10, 50)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Ball Speed: %f", g.ball.speedX), 10, 70)
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// start listening to bluetoothctl in a goroutine
	go monitorVolume()

	// start Ebiten game loop
	g := &Game{}

	// initialize ball
	g.ball.x = screenWidth / 2
	g.ball.y = screenHeight / 2
	g.ball.speedX = 2
	g.ball.speedY = 1 + rand.Float32()*2

	// initialize player1 paddle
	g.player1.x = 20
	g.player1.y = screenHeight - paddleHeight

	// initialize player2 paddle
	g.player2.x = screenWidth - paddleWidth - 20
	g.player2.y = screenHeight - paddleHeight

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// monitorVolume spawns bluetoothctl and parses Volume changes
func monitorVolume() {
	// Connect to the system bus
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatalf("Failed to connect to system bus: %v", err)
	}

	// Add a rule to only capture org.bluez signals
	call := conn.BusObject().Call(
		"org.freedesktop.DBus.AddMatch", 0,
		"type='signal',sender='org.bluez'",
	)
	if call.Err != nil {
		log.Fatalf("Failed to add match rule: %v", call.Err)
	}

	fmt.Println("Listening for org.bluez signals...")

	// Receive and handle signals
	ch := make(chan *dbus.Signal, 10)
	conn.Signal(ch)
	for signal := range ch {
		if signal.Name == "org.freedesktop.DBus.Properties.PropertiesChanged" {
			iface := signal.Body[0].(string)
			changedProps := signal.Body[1].(map[string]dbus.Variant)

			if iface == "org.bluez.MediaTransport1" {
				if volVariant, ok := changedProps["Volume"]; ok {
					newVolume := volVariant.Value().(uint16)
					// fmt.Printf("Current Volume: %d\n", newVolume)
					atomic.StoreUint32(&volume, uint32(newVolume))
				}
			}
		}
	}
}

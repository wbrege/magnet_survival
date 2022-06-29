package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/kvartborg/vector"
	"github.com/solarlune/resolv"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"time"
)

//go:embed resources/magnet_man.png
var charImgB []byte

const charHeight = 29
const charWidth = 13

const charImgBorder = 1
const charImgWidth = 32
const charImgHeight = 32
const charFrameLength = 6
const charIdleIdx = 0
const charWalkRightIdx = 1
const charWalkLeftIdx = 2
const charWalkDownIdx = 3
const charWalkUpIdx = 4

//go:embed resources/magnet_weapon.png
var wepImgB []byte

//go:embed resources/magnet_heart.png
var heartImgB []byte

type Player struct {
	obj        *resolv.Object
	img        *ebiten.Image
	health     int
	invincible bool
	invincTs   time.Time
	dirVec     vector.Vector
	frameCount int
}

func (p *Player) pos() vector.Vector {
	return vector.Vector{p.obj.X, p.obj.Y}
}

func (p *Player) Update() {
	velocityMagnitude := 1.0
	xVelocity := 0.0
	yVelocity := 0.0

	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		xVelocity += velocityMagnitude
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		xVelocity -= velocityMagnitude
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		yVelocity -= velocityMagnitude
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		yVelocity += velocityMagnitude
	}

	p.dirVec = vector.Vector{xVelocity, yVelocity}.Unit()
	p.obj.X += p.dirVec.X()
	p.obj.Y += p.dirVec.Y()
	p.frameCount++

	p.obj.Update()
	if collision := p.obj.Check(0, 0); collision != nil && !p.invincible {
		p.health -= 1
		p.invincible = true
		p.invincTs = time.Now()
	}

	if p.invincible && (time.Now().After(p.invincTs.Add(5 * time.Second))) {
		p.invincible = false
	}
}

func (p *Player) Draw(screen *ebiten.Image, op *ebiten.DrawImageOptions) {
	frameIdx := (p.frameCount / 5) % charFrameLength
	idx := charIdleIdx
	if p.dirVec.X() > 0 {
		idx = charWalkRightIdx
	} else if p.dirVec.X() < 0 {
		idx = charWalkLeftIdx
	} else if p.dirVec.Y() < 0 {
		idx = charWalkUpIdx
	} else if p.dirVec.Y() > 0 {
		idx = charWalkDownIdx
	}

	sx, sy := (frameIdx*charImgWidth)+(frameIdx*charImgBorder), (idx*charImgHeight)+(idx*charImgBorder)
	//ebitenutil.DebugPrintAt(screen, fmt.Sprintf("sx: %v, sy: %v", sx, sy), 50, 50)

	isInvincibleFrame := p.invincible && ((p.frameCount/10)%2 == 0)

	r := image.Rect(sx, sy, sx+charImgWidth, sy+charImgHeight)
	img := p.img.SubImage(r).(*ebiten.Image)
	if !isInvincibleFrame {
		screen.DrawImage(img, op)
	}
}

type Weapon struct {
	img               *ebiten.Image
	obj               *resolv.Object
	width             float64
	height            float64
	targetPos         vector.Vector
	arrived           bool
	acceleration      float64
	velocity          float64
	angle             float64
	targetAngle       float64
	angularVelocity   float64
	dirVec            vector.Vector
	startPos          vector.Vector
	quarterPoint      vector.Vector
	threeQuarterPoint vector.Vector
}

func (w *Weapon) pos() vector.Vector {
	return vector.Vector{w.obj.X, w.obj.Y}
}

func (w *Weapon) UpdateTarget(p Player) {
	minMag := 75.0
	maxMag := 100.0

	diffVec := p.pos().Sub(w.pos())

	var distVec vector.Vector
	if diffVec.Magnitude() <= minMag {
		distVec = diffVec.Add(diffVec.Unit().Scale(minMag))
	} else if diffVec.Magnitude() >= maxMag {
		distVec = diffVec.Add(diffVec.Unit().Scale(maxMag))
	} else {
		distVec = diffVec.Scale(2.0)
	}

	unitVec := distVec.Unit()
	newDirVec := vector.Vector{unitVec.X(), unitVec.Y(), 0.0}
	theta := math.Acos(w.dirVec.Dot(newDirVec))

	w.startPos = w.pos()
	w.quarterPoint = distVec.Scale(0.25)
	w.threeQuarterPoint = distVec.Scale(0.75)
	w.targetPos = w.pos().Add(distVec)

	cross, err := w.dirVec.Cross(newDirVec)
	if err != nil {
		// Only throws if vector is not 3D
		log.Fatal(err)
	}

	// Determine if angle is clockwise or not
	angleDiff := theta
	if cross.Z() < 0 {
		angleDiff = 3.14 + (3.14 - theta)
	}

	w.targetAngle = w.angle + angleDiff
	w.dirVec = newDirVec

	w.angularVelocity = angleDiff / w.quarterPoint.Magnitude()
}

func (w *Weapon) UpdatePosition() {
	if w.pos().Sub(w.startPos).Magnitude() < w.quarterPoint.Magnitude() {
		w.velocity += w.acceleration
	} else if w.pos().Sub(w.startPos).Magnitude() >= w.threeQuarterPoint.Magnitude() {
		w.velocity -= w.acceleration
	}

	diffVec := w.targetPos.Sub(w.pos())
	moveVec := diffVec.Unit().Scale(w.velocity)

	if diffVec.Magnitude() < 2.0 {
		w.obj.X = w.targetPos.X()
		w.obj.Y = w.targetPos.Y()
	} else {
		w.obj.X += moveVec.X()
		w.obj.Y += moveVec.Y()
	}

	if math.Abs(w.angle-w.targetAngle) < math.Abs(2*w.angularVelocity) {
		w.angle = w.targetAngle
	}

	if w.angle != w.targetAngle {
		w.angle += w.angularVelocity
	}
}

func (w *Weapon) Update(p Player) {
	if w.arrived {
		w.arrived = false
		w.UpdateTarget(p)
	}

	w.UpdatePosition()
	if w.pos().Equal(w.targetPos) {
		w.arrived = true
	}
	w.obj.Update()
}

type Game struct {
	inited    bool
	width     int
	height    int
	space     *resolv.Space
	player    *Player
	weapon    *Weapon
	heartImg  *ebiten.Image
	gamestart time.Time
	gameover  bool
}

func (g *Game) init() {
	g.space = resolv.NewSpace(g.width, g.height, 1, 1)

	charImg, _, err := image.Decode(bytes.NewReader(charImgB))
	if err != nil {
		log.Fatal(err)
	}
	charImgE := ebiten.NewImageFromImage(charImg)

	playerObj := resolv.NewObject(
		float64(g.width)/2.0-float64(charWidth)/2.0,
		float64(g.height)/2.0-float64(charHeight)/2.0,
		float64(charWidth),
		float64(charHeight),
	)
	playerObj.SetShape(resolv.NewCircle(
		float64(g.width)/2.0-float64(charWidth)/2.0,
		float64(g.height)/2.0-float64(charHeight)/2.0,
		float64(charHeight)/2.0),
	)
	g.space.Add(playerObj)

	g.player = &Player{
		obj:        playerObj,
		img:        charImgE,
		health:     3,
		invincible: false,
	}

	wepImg, _, err := image.Decode(bytes.NewReader(wepImgB))
	if err != nil {
		log.Fatal(err)
	}
	wepImgE := ebiten.NewImageFromImage(wepImg)

	weaponWidth, weaponHeight := wepImgE.Size()
	weaponObj := resolv.NewObject(0.0, float64(g.height)/2.0-float64(weaponHeight)/2.0, float64(weaponWidth), float64(weaponHeight))
	weaponObj.SetShape(resolv.NewCircle(0.0, 100.0, float64(weaponHeight)/2.0))
	weaponObj.AddTags("weapon")
	g.space.Add(weaponObj)

	g.weapon = &Weapon{
		obj:          weaponObj,
		img:          wepImgE,
		width:        float64(weaponWidth),
		height:       float64(weaponHeight),
		arrived:      false,
		acceleration: 0.04,
		velocity:     0.25,
		dirVec:       vector.Vector{0.0, 0.0, 0.0},
	}
	g.weapon.UpdateTarget(*g.player)

	heartImg, _, err := image.Decode(bytes.NewReader(heartImgB))
	if err != nil {
		log.Fatal(err)
	}
	heartImgE := ebiten.NewImageFromImage(heartImg)
	g.heartImg = heartImgE

	g.gamestart = time.Now()
	g.inited = true
	g.gameover = false
}

func (g *Game) Update() error {
	if (!g.inited || g.gameover) && ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.init()
	}

	if !g.gameover && g.inited {
		g.player.Update()
		g.weapon.Update(*g.player)
	}

	if g.inited && g.player.health <= 0 {
		g.gameover = true
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{0x2A, 0xB, 0x32, 0xff})
	if g.inited && !g.gameover {
		op := &ebiten.DrawImageOptions{}

		op.GeoM.Translate(g.player.obj.X, g.player.obj.Y)
		g.player.Draw(screen, op)

		op.GeoM.Reset()
		op.GeoM.Translate(-g.weapon.width/2.0, -g.weapon.height/2.0)
		op.GeoM.Rotate(g.weapon.angle)
		op.GeoM.Translate(g.weapon.obj.X+g.weapon.width/2.0, g.weapon.obj.Y+g.weapon.height/2.0)
		screen.DrawImage(g.weapon.img, op)

		for i := 0; i < g.player.health; i++ {
			op.GeoM.Reset()
			op.GeoM.Translate(10.0+float64(i*35), 10.0)
			screen.DrawImage(g.heartImg, op)
		}
		timePassed := time.Now().Sub(g.gamestart)
		ebitenutil.DebugPrintAt(
			screen,
			fmt.Sprintf(
				"%02v:%02v",
				math.Mod(math.Floor(timePassed.Minutes()), 60),
				math.Mod(math.Floor(timePassed.Seconds()), 60),
			),
			g.width-100,
			10.0,
		)
	} else if g.gameover {
		ebitenutil.DebugPrintAt(screen, "GAME OVER", g.width/2-35, g.height/2-10)
		ebitenutil.DebugPrintAt(screen, "Press Space to restart", g.width/2-80, g.height/2+60)
	} else {
		ebitenutil.DebugPrintAt(screen, "UNTITLED MAGNET BASED\n    SURVIVAL GAME", g.width/2-75, g.height/2-50)
		ebitenutil.DebugPrintAt(screen, "Dodge the big magnetic wrecking ball coming after you!\n            For as long as you can...", g.width/2-165, g.height/2)
		ebitenutil.DebugPrintAt(screen, "Arrow Keys to move", g.width/2-70, g.height/2+40)
		ebitenutil.DebugPrintAt(screen, "Press Space to start", g.width/2-75, g.height/2+60)
	}

	//g.DebugVector(screen)
	//ebitenutil.DebugPrint(screen, fmt.Sprintf("Angle: %d", g.weapon.angle))
	//g.DebugDraw(screen, g.space)
	//ebitenutil.DebugPrint(screen, fmt.Sprintf("Health: %d, Collision: %d", g.player.health, g.player.obj.Check(0, 0) == nil))
	//ebitenutil.DebugPrint(screen,
	//	fmt.Sprintf("playerX: %d, playerY: %d\ntargetX: %d, targetY: %d",
	//		g.player.obj.X, g.player.obj.Y, g.weapon.targetPos.X(), g.weapon.targetPos.Y()))
}

func (g *Game) DebugVector(screen *ebiten.Image) {
	drawColor := color.RGBA{255, 255, 0, 255}

	ebitenutil.DrawLine(
		screen,
		g.weapon.startPos.X(),
		g.weapon.startPos.Y(),
		g.weapon.targetPos.X(),
		g.weapon.targetPos.Y(),
		drawColor,
	)
}

func (g *Game) DebugDraw(screen *ebiten.Image, space *resolv.Space) {
	for y := 0; y < space.Height(); y++ {
		for x := 0; x < space.Width(); x++ {

			cell := space.Cell(x, y)

			cw := float64(space.CellWidth)
			ch := float64(space.CellHeight)
			cx := float64(cell.X) * cw
			cy := float64(cell.Y) * ch

			drawColor := color.RGBA{20, 20, 20, 255}

			if cell.Occupied() {
				drawColor = color.RGBA{255, 255, 0, 255}
			}

			ebitenutil.DrawLine(screen, cx, cy, cx+cw, cy, drawColor)
			ebitenutil.DrawLine(screen, cx+cw, cy, cx+cw, cy+ch, drawColor)
			ebitenutil.DrawLine(screen, cx+cw, cy+ch, cx, cy+ch, drawColor)
			ebitenutil.DrawLine(screen, cx, cy+ch, cx, cy, drawColor)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.width, g.height
}

func NewGame() *Game {
	return &Game{
		width:  480,
		height: 360,
	}
}

func main() {
	g := NewGame()
	ebiten.SetWindowSize(g.width*2, g.height*2)
	ebiten.SetWindowTitle("Untitled Magnet Based Survival Game")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

# Untitled Magnet Based Survival Game
Very basic survival game built using [Ebitengine](https://ebiten.org/) for the [Ebiten Game Jam](https://itch.io/jam/ebiten-game-jam).

First time using this engine, haven't bothered to factor it nicely since I was just learning how the engine works,
so presently it's all just one big spaghetti code in `main.go`.

## How to run

Run locally using `go run ./main.go`.

Compile a wasm binary using `GOOS=js GOARCH=wasm go build -o magnet_survival.wasm github.com/wbrege/magnet_survival`.
Slap the binary, `index.html`, `resources` folder, and `wasm_exec.js` into a zip folder and you can upload it to itch.io.

## TODO list for myself
- [x] Basic survival game that works
- [ ] Music
- [ ] Enemies
- [ ] Power ups
- [ ] Better goal
- [ ] Stretch goal refactor the spaghetti

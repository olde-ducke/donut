package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

const thetaSpacing float64 = 0.07
const phiSpacing float64 = 0.02

var R1, R2, K1, K2, ratio float64
var width, height int
var target time.Duration
var debug bool
var frameTime, renderTime, displayTime, getSizeTime time.Duration

var chars = []byte{'.', ',', '-', '~', ':', ';', '=', '!', '*', '#', '$', '@'}

func run() (int, error) {
	// hide cursor
	fmt.Print("\033[?25l")
	// unhide on exit
	defer fmt.Print("\033[?25h")

	// get terminal descriptor
	descriptor := int(os.Stdin.Fd())
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM, syscall.SIGINT)

	A := 1.0
	B := 1.0

	var start, render, display, getSize time.Time
	var err error
	for {
		start = time.Now()

		getSize = time.Now()
		width, height, err = terminal.GetSize(descriptor)
		if err != nil {
			return 2, err
		}

		K1 = float64(height) * K2 * 3 / (8 * (R1 + R2))
		getSizeTime = time.Since(getSize)

		render = time.Now()
		frame := renderFrame(A, B)
		if debug {
			addDebugInfo(frame)
		}
		renderTime = time.Since(render)

		display = time.Now()
		displayFrame(frame)
		displayTime = time.Since(display)

		A += thetaSpacing
		B += phiSpacing

		select {
		case sig := <-sigTerm:
			return 1, fmt.Errorf("%v", sig)
		default:
			time.Sleep(target - getSizeTime - renderTime - displayTime - 100*time.Microsecond)
		}

		frameTime = time.Since(start)
	}

	return 0, nil
}

func renderFrame(A, B float64) [][]byte {
	// precompute sines and cosines of A and B
	cosA := math.Cos(A)
	sinA := math.Sin(A)
	cosB := math.Cos(B)
	sinB := math.Sin(B)
	offX := float64(width) * 0.5
	offY := float64(height) * 0.5

	output := make([][]byte, height)
	zBuffer := make([][]float64, height)
	for x := range output {
		zBuffer[x] = make([]float64, width)
		output[x] = make([]byte, 0, width)
		for y := 0; y < width; y++ {
			output[x] = append(output[x], ' ')
		}
	}

	// theta goes around the cross-sectional circle of a torus
	for theta := 0.0; theta < 2*math.Pi; theta += thetaSpacing {
		// precompute sines and cosines of theta
		cosTheta := math.Cos(theta)
		sinTheta := math.Sin(theta)

		// phi goes around the center of revolution of a torus
		for phi := 0.0; phi < 2*math.Pi; phi += phiSpacing {
			// precompute sines and cosines of phi
			cosPhi := math.Cos(phi)
			sinPhi := math.Sin(phi)

			// the x,y coordinate of the circle, before revolving (factored
			// out of the above equations)
			circleX := R2 + R1*cosTheta
			circleY := R1 * sinTheta

			// final 3D (x,y,z) coordinate after rotations, directly from
			// our math above
			x := circleX*(cosB*cosPhi+sinA*sinB*sinPhi) -
				circleY*cosA*sinB
			y := circleX*(sinB*cosPhi-sinA*cosB*sinPhi) +
				circleY*cosA*cosB
			z := K2 + cosA*circleX*sinPhi + circleY*sinA
			ooz := 1 / z

			// x and y projection.  note that y is negated here, because y
			// goes up in 3D space but down on 2D displays.
			xp := int(offX + K1*ooz*x*ratio + offX*sinA)
			yp := int(offY - K1*ooz*y + offY*cosB)

			// calculate luminance.  ugly, but correct.
			L := cosPhi*cosTheta*sinB - cosA*cosTheta*sinPhi -
				sinA*sinTheta + cosB*(cosA*sinTheta-cosTheta*sinA*sinPhi)

			// L ranges from -sqrt(2) to +sqrt(2).  If it's < 0, the surface
			// is pointing away from us, so we won't bother trying to plot it.
			if L > 0 && yp >= 0 && xp >= 0 && xp < width && yp < height {
				if ooz > zBuffer[yp][xp] {
					zBuffer[yp][xp] = ooz
					luminanceIdx := int(L * 8)

					// luminance_index is now in the range 0..11 (8*sqrt(2) = 11.3)
					// now we lookup the character corresponding to the
					// luminance and plot it in our output:
					output[yp][xp] = chars[luminanceIdx]
				}
			}
		}
	}
	return output
}

func displayFrame(output [][]byte) {
	// now, dump output[] to the screen.
	// bring cursor to "home" location, in just about any currently-used
	// terminal emulation mode
	fmt.Print("\x1b[H")
	for j := range output {
		fmt.Print(string(output[j]))
	}
}

func addDebugInfo(frame [][]byte) {
	var decoration string
	if frameTime > target {
		decoration = "\x1b[31;1m"
	} else {
		decoration = "\x1b[32;1m"
	}

	debugInfo := []string{
		fmt.Sprintf("last frame:    %s%s\x1b[0m", decoration, frameTime),
		fmt.Sprintf("display time:  %s", displayTime),
		fmt.Sprintf("render time:   %s", renderTime),
		fmt.Sprintf("get size time: %s", getSizeTime),
	}

	for i, j := 0, len(frame)-1; i < 4 && j >= 0; j-- {
		for x, r := range debugInfo[i] {
			if x >= len(frame[0]) {
				break
			}

			frame[j][x] = byte(r)
		}
		i++
	}
}

func main() {
	flag.Usage = func() {
		fmt.Printf(`
draws spinning 3D donut in ascii
press <Ctrl>+C to exit

usage: %s [<arguments>]

arguments:
  --r1     - TBD                        default: 1.0
  --r2     - TBD                        default: 2.0
  --k2     - distance from the viewer   default: 5.0
  --ratio  - height to width ratio      default: 2.0
  --target - target frame time          default: 33.333ms
  --debug  - print debug info to output
`, os.Args[0])
	}
	flag.Float64Var(&R1, "r1", 1.0, "")
	flag.Float64Var(&R2, "r2", 2.0, "")
	flag.Float64Var(&K2, "k2", 5.0, "")
	flag.Float64Var(&ratio, "ratio", 2.0, "")
	flag.DurationVar(&target, "target", 33333*time.Microsecond, "")
	flag.BoolVar(&debug, "debug", false, "")
	flag.Parse()

	n, err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(n)
	}
}

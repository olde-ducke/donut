package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

const thetaSpacing float64 = 0.07
const phiSpacing float64 = 0.02
const R1 float64 = 1.0
const R2 float64 = 2.0
const K2 float64 = 5.0

var K1 float64
var width, height int

var ratio = 2.0
var chars = []rune{'.', ',', '-', '~', ':', ';', '=', '!', '*', '#', '$', '@'}

func run() (int, error) {
	A := 1.0
	B := 1.0
	var err error

	for {
		width, height, err = terminal.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			return 1, err
		}

		K1 = float64(height) * K2 * 3 / (8 * (R1 + R2))

		renderFrame(A, B)
		A += thetaSpacing
		B += phiSpacing

		time.Sleep(33 * time.Millisecond)
	}

	return 0, nil
}

func renderFrame(A, B float64) {
	// precompute sines and cosines of A and B
	cosA := math.Cos(A)
	sinA := math.Sin(A)
	cosB := math.Cos(B)
	sinB := math.Sin(B)

	output := make([][]rune, height)
	zBuffer := make([][]float64, height)
	for x := range output {
		zBuffer[x] = make([]float64, width)
		output[x] = make([]rune, 0, width)
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
			xp := int(float64(width)*0.5 + K1*ooz*x*ratio)
			yp := int(float64(height)*0.5 - K1*ooz*y)

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

	// now, dump output[] to the screen.
	// bring cursor to "home" location, in just about any currently-used
	// terminal emulation mode
	fmt.Print("\x1b[H")
	for j := range output {
		fmt.Println(string(output[j]))
	}
}

func main() {
	n, err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(n)
	}

	fmt.Println(n)
}

package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"github.com/urfave/cli"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type Data struct {
	Total     int
	TotalTime float64
	Points    []E6Pos
}

//E6POS data structure
type E6Pos struct {
	Index    int     //used to indicate the point number
	TimeCode float64 //time count from beginning of spline
	TimeMark float64 //time since last time mark
	X        float64
	Y        float64
	Z        float64
	A        float64
	B        float64
	C        float64
	S        int
	T        int
	E1       float64
	E2       float64
	E3       float64
	E4       float64
	E5       float64
	E6       float64
}

//a simple 3 diamensional point
type point struct {
	P1 float64
	P2 float64
	P3 float64
}

var ticker *time.Ticker

func main() {

	ticker = time.NewTicker(time.Millisecond * 500)

	app := cli.NewApp()
	app.Name = "GoKuka"
	app.Version = "0.1"
	app.Author = "Noah Shibley"
	app.Email = "noah@socialhardware.net"
	app.Usage = "A Kuka Robot Language Generator"
	app.Action = func(c *cli.Context) {
		cli.ShowAppHelp(c)
	}

	app.Commands = []cli.Command{
		{
			Name:      "random",
			ShortName: "r",
			Flags: []cli.Flag{

				cli.IntFlag{
					Name:  "numberOf, n",
					Value: 30,
					Usage: "Set the number of random circles",
				},
			},
			Usage:       "random -n 40",
			Description: "Generate random circle spline paths",
			Action:      randomCircle,
		},
		{
			Name:      "repeat",
			ShortName: "p",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "numberOf, n",
					Value: 30,
					Usage: "Set the number of random circles",
				},
			},
			Usage:       "repeat -n 30",
			Description: "generate n circles of the same size",
			Action:      repeatCircle,
		},
		{
			Name:      "csv",
			ShortName: "c",
			Flags: []cli.Flag{

				cli.IntFlag{
					Name:  "max, m",
					Value: 0,
					Usage: "Sets the max number of points per file",
				},
			},
			Usage:       "csv `FILE` -m 50",
			Description: "reads a spline point csv from `FILE`",
			Action:      splineFile,
		},
	}

	app.Run(os.Args)

}

//splineFile
//Parses a csv file in the following format
//Time	  X		   Y	   Z		  A		  B		  C
//1.0000, 44.9624, 8.7501, 1119.9937, 9.2796, 0.0000, 0.0000
func splineFile(c *cli.Context) {

	outFileName := "fileSpline"
	fileName := ""
	points := make([]E6Pos, 0)
	maxPoints := 0

	if c.NArg() == 1 {
		fileName = c.Args().First()
	} else {
		cli.ShowCommandHelp(c, "csv")
	}

	maxPoints = c.Int("max") //-m or max flag

	f, err := os.Open(fileName)
	defer f.Close()
	if err != nil {
		fmt.Errorf("cannot open file: %s", err)
		return
	}

	csvReader := csv.NewReader(bufio.NewReader(f))

	fmt.Printf("Generating...\n")
	go waitIndicate() //print a . every 500ms to indicate time passing

	parsePoints(csvReader, &points)

	if maxPoints == 0 { //default is 0 if not set in flag
		maxPoints = len(points) //set to the whole file
	}

	splitValue := (float64(len(points)) / float64(maxPoints))
	numFiles := math.Ceil(float64(splitValue)) //always round up
	fmt.Printf("number of files: %d \n", int(numFiles))

	for i := 0; i < int(numFiles); i++ {

		startIndex := i * maxPoints
		endIndex := startIndex + maxPoints
		outPoints := points[startIndex:endIndex]

		if endIndex >= len(points) { //if the end of the spline just add the remainer points to a new file
			outPoints = points[startIndex:]
		}

		writeTemplate(outPoints, "kukaSplineTemplate.dat", fmt.Sprintf("%s%d.dat", outFileName, i)) //dat
		writeTemplate(outPoints, "kukaSplineTemplate.src", fmt.Sprintf("%s%d.src", outFileName, i)) //src

		fmt.Printf("Files saved as: %s%d.dat & %s%d.src\n", outFileName, i, outFileName, i)
	}

	ticker.Stop()
	fmt.Printf("Done\n")

}

func repeatCircle(c *cli.Context) {

	xyzOffset := &point{700.0, 0.0, 600.0}
	abcAngles := &point{90.0, 0.0, -180.0}

	numOfCircles := 30

	if c.NArg() > 0 {

		numOfCircles = c.Int("numberOf")
	}

	pointIndex := 0

	points := circle(100.0, xyzOffset, abcAngles, &pointIndex)

	writeTemplate(points, "kukaSplineTemplate.dat", "repeatCircleSpline.dat") //dat

	//draw multiple circles
	totalPoints := make([]E6Pos, 0)
	for i := 0; i < numOfCircles; i++ {
		totalPoints = append(totalPoints, points...)
	}

	writeTemplate(totalPoints, "kukaSplineTemplate.src", "repeatCircleSpline.src") //src
}

func randomCircle(c *cli.Context) {

	points := make([]E6Pos, 0)
	numOfCircles := 30

	if c.NArg() > 0 {
		numOfCircles = c.Int("numberOf")
	}
	pointIndex := 0

	for i := 0; i < numOfCircles; i++ {

		x := float64(rand.Intn(100) + 550.0)
		z := float64(rand.Intn(100) + 450.0)

		xyzOffset := &point{x, 0.0, z}

		a := float64(rand.Intn(45) + 45)

		abcAngles := &point{a, 0.0, -180.0}

		radius := float64(rand.Intn(100) + 30)
		p := circle(radius, xyzOffset, abcAngles, &pointIndex)
		points = append(points, p...)
	}

	writeTemplate(points, "kukaSplineTemplate.dat", "randomCircleSpline.dat") //dat
	writeTemplate(points, "kukaSplineTemplate.src", "randomCircleSpline.src") //src
}

//circle
func circle(radius float64, center *point, abcAngles *point, index *int) []E6Pos {

	points := make([]E6Pos, 0)

	angleStep := 0.1
	angle := 0.0
	//index := 0

	for angle < 2*math.Pi {
		*index++
		y, z := math.Sincos(angle)
		y = (y * radius) + center.P2
		z = (z * radius) + center.P3

		x := center.P1

		angle += angleStep

		circlePoint := &E6Pos{Index: *index, X: x, Y: y, Z: z, A: abcAngles.P1, B: abcAngles.P2, C: abcAngles.P3, S: 2, T: 43, E1: 0.0, E2: 0.0, E3: 0.0, E4: 0.0, E5: 0.0, E6: 0.0}

		points = append(points, *circlePoint)

		//fmt.Printf("{%v}\n", circlePoint)
	}

	return points
}

//waitIndicate
func waitIndicate() {
	for _ = range ticker.C {
		fmt.Println(".")
	}
}

//parsePoints
func parsePoints(csvReader *csv.Reader, points *[]E6Pos) {

	currentRec := 1

	for {
		record, err := csvReader.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}

		p := &E6Pos{}

		p.Index = currentRec

		p.TimeCode, _ = strconv.ParseFloat(strings.TrimSpace(record[0]), 64)

		p.X, _ = strconv.ParseFloat(strings.TrimSpace(record[1]), 64)
		p.Y, _ = strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
		p.Z, err = strconv.ParseFloat(strings.TrimSpace(record[3]), 64)

		if err != nil {
			fmt.Println(err)
		}

		p.A, _ = strconv.ParseFloat(strings.TrimSpace(record[4]), 64)
		p.B, _ = strconv.ParseFloat(strings.TrimSpace(record[5]), 64)
		p.C, _ = strconv.ParseFloat(strings.TrimSpace(record[6]), 64)
		p.TimeMark, _ = strconv.ParseFloat(strings.TrimSpace(record[7]), 64)

		p.S = 6
		p.T = 19

		//fmt.Printf("%v\n", record)
		*points = append(*points, *p) //pushback
		currentRec++
		// Display record.

	}

}

//writeTemplate
func writeTemplate(posData []E6Pos, templateFile string, outFile string) {

	outData := &Data{Total: len(posData), TotalTime: posData[len(posData)-1].TimeCode, Points: posData}

	tmpl, err := template.New(templateFile).
		Funcs(template.FuncMap{"mod": func(i, j int) bool { return i%j == 0 }}).                         //add a mod function to the template
		Funcs(template.FuncMap{"last": func(i int) bool { return i%outData.Total == 0 }}).               //add a last in range to the template
		Funcs(template.FuncMap{"first": func(i int) bool { return i%outData.Total == 1 }}).              //add a first in range to the template
		Funcs(template.FuncMap{"fTotal": func() string { return fmt.Sprintf("%-23d", outData.Total) }}). //returns a total value of fixed length
		Funcs(template.FuncMap{"pTotal": func() string { return fmt.Sprintf("%-19d", outData.Total) }}). //returns a total value of fixed length
		Funcs(template.FuncMap{"totalTime": func() string {                                              //return total time at end of spline block

			totalTime := 0.0
			for _, p := range posData {
				totalTime = totalTime + p.TimeMark
			}

			rTime := calcRemainTime(posData)
			return fmt.Sprintf("%f", totalTime+rTime)

		}}).
		Funcs(template.FuncMap{"rTime": func() string { //return remaining time for last time part in spline
			return fmt.Sprintf("%f", calcRemainTime(posData))
		}}).
		Funcs(template.FuncMap{"rMarkTest": func(i int, mark float64) bool { //test if time mark already exists or if end of spline

			if mark == 0 && i%outData.Total == 0 { //no mark & index is mod of last point in spline
				return true
			}
			return false
		}}).
		Funcs(template.FuncMap{"firstIndex": func() int { //returns first index of spline slice
			return posData[0].Index
		}}).
		Delims("[[", "]]").
		ParseFiles(templateFile)
	if err != nil {
		fmt.Println(err)
	}

	f, err := os.Create(outFile)
	if err != nil {
		fmt.Println("create file: ", err)
	}

	err = tmpl.Execute(f, outData)
	if err != nil {
		fmt.Println(err)
	}

}

//calcRemainTime
func calcRemainTime(posData []E6Pos) float64 {

	var prevPartTime float64
	//walk back till the prev time mark
	for i := len(posData) - 1; i > 0; i-- {
		prevPartTime = posData[i].TimeCode
		if posData[i].TimeMark != 0 {
			break
		}
	}
	//subtract the total time from the last time_part time
	return posData[len(posData)-1].TimeCode - prevPartTime

}

package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/urfave/cli/v2" // imports as package "cli"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

func GetSanitizedFileName(file string) string {
	tuple := strings.Split(file, ".")
	if len(tuple) != 2 {
		panic(fmt.Sprintf("file has multiple periods: %s", file))
	}
	return fmt.Sprintf("%s_sanitized.%s", tuple[0], tuple[1])
}

type Cell interface {
	String() string
	ContainsPersonalData() bool
}

type Record interface {
	Cells() []Cell
	Raw() []string
	ContainsPersonalData() bool
	Timestamp() time.Time
	Actor() string
	Action() string
}

type RecordIO interface {
	WriteCSV(filename string, records []Record) error
	ReadCSV(filename string) ([]Record, error)
}

func NewRecordIO() RecordIO {
	return recordio{}
}

type recordio struct{}

func (r recordio) ReadCSV(filename string) ([]Record, error) {
	data, err := ioutil.ReadFile(filename)
	Log(fmt.Sprintf("%d", len(data)))

	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(bytes.NewReader(data))
	rawRecords, err := reader.ReadAll()
	Log(fmt.Sprintf("%d", len(rawRecords)))
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(rawRecords))
	for _, rawRecord := range rawRecords[1:] {
		records = append(records, NewRecord(rawRecord))
	}
	return records, nil
}

func (r recordio) WriteCSV(filename string, records []Record) error {
	buffer := bytes.NewBuffer([]byte{})
	writer := csv.NewWriter(buffer)

	for _, record := range records {
		writer.Write(record.Raw())
	}
	return ioutil.WriteFile(filename, buffer.Bytes(), 775)
}

type genericrecord struct {
	cells []Cell
}

func (g *genericrecord) Cells() []Cell {
	return g.cells
}

func (g *genericrecord) Raw() []string {
	out := make([]string, 0, len(g.cells))
	for _, cell := range g.cells {
		out = append(out, cell.String())
	}
	return out
}

type stringcell string

func (s stringcell) String() string {
	return string(s)
}

func (s stringcell) ContainsPersonalData() bool {
	personalDataKeys := []string{"Your hand"}
	for _, key := range personalDataKeys {
		if strings.Contains(string(s), key) {
			return true
		}
	}
	return false
}

func (g *genericrecord) ContainsPersonalData() bool {
	for _, cell := range g.cells {
		if cell.ContainsPersonalData() {
			return true
		}
	}
	return false
}

func (g *genericrecord) Timestamp() time.Time {
	s := g.Cells()[1].String()
	Log(s)
	time, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return time
}

func (g *genericrecord) Actor() string {
	cell := strings.Split(g.Cells()[0].String(), "@")[0]
	cell = strings.Trim(cell, "\"")
	return strings.Trim(cell, " ")
}

func (g *genericrecord) Action() string {
	cell := g.Cells()[0]
	split := strings.Split(cell.String(), "\"")
	return split[len(split)-1]
}

func NewRecord(records []string) Record {
	out := &genericrecord{
		cells: make([]Cell, 0, len(records)),
	}
	for _, cell := range records {
		out.cells = append(out.cells, stringcell(cell))
	}
	return out
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "File to Parse",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output file",
			},
		},
		Name:  "anonymize",
		Usage: "remove hands",
		Action: func(c *cli.Context) error {
			filename := c.String("file")
			output := c.String("output")
			Log(fmt.Sprintf("anonimizing file '%s'", filename))
			io := NewRecordIO()

			records, err := io.ReadCSV(filename)
			Log(fmt.Sprintf("%d", len(records)))
			if err != nil {
				return err
			}
			if len(records) == 0 {
				Log("no records to santize, exiting")
				return nil
			}

			santitized := make([]Record, 0, len(records))
			for _, record := range records {
				if record.ContainsPersonalData() {
					Log(fmt.Sprintf("discarding unsanitized row: \"%s\"", record.Raw()))
					continue
				}
				santitized = append(santitized, record)
			}
			return io.WriteCSV(output, santitized)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func Log(s string) {
	fmt.Printf("%s\n", s)
}

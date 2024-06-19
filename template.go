package template

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"text/template"
)

var (
	ErrInvalidFile     = errors.New("invalid file")
	ErrInvalidCSV      = errors.New("invalid csv")
	ErrInvalidTemplate = errors.New("invalid template")
)

type FILE struct {
	name, mimetype string
	b              []byte
}

func NewFile(name, mimetype string, b []byte) (*FILE, error) {
	if name = strings.TrimSpace(name); len(name) == 0 {
		return nil, fmt.Errorf("%w: empty filename", ErrInvalidFile)
	} else if mimetype = strings.TrimSpace(mimetype); len(mimetype) == 0 {
		return nil, fmt.Errorf("%w: empty mimetype", ErrInvalidFile)
	}
	return &FILE{
		name:     strings.Clone(name),
		mimetype: strings.Clone(mimetype),
		b:        bytes.Clone(b),
	}, nil
}

type header_field struct {
	key   string
	index int
}

type CSV_HEADER struct {
	header        []string
	sorted_header []header_field
}

func (c *CSV_HEADER) Len() int { return len(c.header) }

func (c *CSV_HEADER) FindKeyIndex(key string) int {
	if c.sorted_header == nil {
		c.sorted_header = make([]header_field, 0, len(c.header))
		for index, key := range c.header {
			c.sorted_header = append(c.sorted_header, header_field{key, index})
		}
		sort.Slice(c.sorted_header, func(i, j int) bool { return c.sorted_header[i].key < c.sorted_header[j].key })
	}
	if i := sort.Search(len(c.sorted_header), func(i int) bool { return c.sorted_header[i].key >= key }); i == -1 || c.sorted_header[i].key != key {
		return -1
	} else {
		return c.sorted_header[i].index
	}
}

type CSV struct {
	*FILE
	header CSV_HEADER
	rows   [][]string
}

func NewCSV(name, mimetype string, b []byte) (*CSV, error) {
	file, err := NewFile(name, mimetype, b)
	if err != nil {
		return nil, err
	}
	results := &CSV{
		FILE: file,
	}
	csv := csv.NewReader(bytes.NewReader(file.b))
	// read header row
	header, err := csv.Read()
	if err != nil {
		return nil, fmt.Errorf("%w: error reading header row %w", ErrInvalidCSV, err)
	}
	results.header = CSV_HEADER{header: slices.Clone(header)}
	rows, err := csv.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: error reading csv body: %w", ErrInvalidCSV, err)
	}
	for _, row := range rows {
		if want, got := results.header.Len(), len(row); want != got {
			return nil, fmt.Errorf("%w: malformed body (row has %d fields; header indicates %d)", ErrInvalidCSV, got, want)
		} else {
			results.rows = append(results.rows, slices.Clone(row))
		}
	}
	return results, nil
}

// holds a raw copy of the template it was given
// and an updated one that works with the given CSV
type Template struct {
	raw_file             *FILE
	csv                  *CSV
	raw_adapted_template []byte
	adapted_template     *template.Template
	position             int
}

func (t *Template) Next() (templateoutput []byte, csvrow []string, err error) {
	index := t.position
	if index >= len(t.csv.rows) {
		err = fmt.Errorf("%w: nothing else to do", io.EOF)
		return
	}
	t.position++ // move position forward
	csvrow = slices.Clone(t.csv.rows[index])
	writer := new(bytes.Buffer)
	err = t.adapted_template.Execute(writer, csvrow)
	templateoutput = writer.Bytes()
	return
}

func (t *Template) ExecuteAll() (templateoutput [][]byte, rows [][]string, err error) {
	for {
		output, row, err := t.Next()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return templateoutput, rows, err
			}
			return templateoutput, rows, nil
		}
		templateoutput = append(templateoutput, output)
		rows = append(rows, row)
	}
}

var (
	errNoToken = errors.New("no token found")
)

func readUntilNextTemplateActionEnd(r *bufio.Reader) (string, error) {
	var buffer strings.Builder
	for {
		b, err := r.ReadByte()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return buffer.String(), fmt.Errorf("%w while reading", err)
			}
			return buffer.String(), fmt.Errorf("%w: found %w before template action end", errNoToken, err)
		}
		buffer.WriteByte(b)
		if tmp := buffer.String(); strings.HasSuffix(tmp, "}}") {
			return buffer.String(), nil
		}
	}
}

func readUntilNextTemplateAction(r *bufio.Reader) (string, error) {
	var buffer strings.Builder
	for {
		b, err := r.ReadByte()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return buffer.String(), fmt.Errorf("%w while reading", err)
			}
			return buffer.String(), fmt.Errorf("%w: found %w before template action", errNoToken, err)
		}
		buffer.WriteByte(b)
		if tmp := buffer.String(); strings.HasSuffix(tmp, "{{") {
			return buffer.String(), nil
		}
	}
}

func AdaptTemplateToCSV(csv *CSV, raw_template *FILE) ([]byte, error) {
	var results bytes.Buffer
	reader := bufio.NewReader(bytes.NewReader(raw_template.b))
	for {
		got, err := readUntilNextTemplateAction(reader)
		results.WriteString(got)
		if err != nil {
			if !errors.Is(err, errNoToken) {
				return nil, fmt.Errorf("%w: error reading template %w", ErrInvalidTemplate, err)
			}
			// this is fine, no more tokens to parse. the rest was in got and is in results.
			return results.Bytes(), nil
		}
		got, err = readUntilNextTemplateActionEnd(reader)
		if err != nil {
			return nil, fmt.Errorf("%w: no action end found after opening {{", ErrInvalidTemplate)
		}
		got = strings.Trim(got, " \t\n.}")
		// got should be a field in the csv header
		i := csv.header.FindKeyIndex(got)
		if i == -1 {
			return nil, fmt.Errorf("%w: key '%s' not found in csv header", ErrInvalidTemplate, got)
		}
		results.WriteString(fmt.Sprintf(" index . %d }}", i))
	}
}

func NewTemplate(name, mimetype string, b []byte, csv *CSV) (*Template, error) {
	file, err := NewFile(name, mimetype, b)
	if err != nil {
		return nil, err
	}
	adapted_template, err := AdaptTemplateToCSV(csv, file)
	if err != nil {
		return nil, err
	}
	template, err := template.New(file.name).Parse(string(adapted_template))
	if err != nil {
		return nil, fmt.Errorf("error creating new text/template.Template: %w", err)
	}
	return &Template{
		raw_file:             file,
		csv:                  csv,
		raw_adapted_template: adapted_template,
		adapted_template:     template,
	}, nil
}

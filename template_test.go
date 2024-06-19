package template

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

const (
	TheTestTemplate = `Hello, {{ RecipientName }}

Welcome to {{ GroupName }}.

Thank you,
{{ SenderName }}
`
	TheTestCSV = `SenderName,RecipientName,GroupName
sender@example.com,recipient@example.com,group name
`
	TheTestCSVTemplateResults = `Hello, recipient@example.com

Welcome to group name.

Thank you,
sender@example.com
`
)

func TestReadCSV(t *testing.T) {
	name := "test.csv"
	mimetype := "text/csv"
	csv, err := NewCSV(name, mimetype, []byte(TheTestCSV))
	if err != nil {
		t.Fatalf("error parsing test.csv: %v\n", err)
	}
	name = "test.template"
	mimetype = "text/plain"
	template, err := NewTemplate(name, mimetype, []byte(TheTestTemplate), csv)
	if err != nil {
		t.Fatalf("error parsing template: %v\n", err)
	}
	results := func() [][]byte {
		results := make([][]byte, 0, 1)
		for {
			got, _, err := template.Next()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					t.Fatalf("error executing: %v\n", err)
				}
				return results
			}
			results = append(results, got)
		}
	}()
	if want, got := 1, len(results); want != got {
		t.Fatalf("error: wanted %d results; got %d", want, got)
	} else if want, got := TheTestCSVTemplateResults, results[0]; !bytes.Equal([]byte(want), got) {
		t.Fatalf("error: wanted %s; got %s\n", want, got)
	}
}

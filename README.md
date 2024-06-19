# package template
package template offers a user-friendly interface to generate documents using Go's text/template package using CSVs. It allows the user to use a template with the csv header field values instead of indexes of row values.

1. parses the .csv data and does some simple validation
    * The CSV parser will refuse to accept CSV with any rows who have a different length than the header row
2. parses the template data and adapts it to work with the CSV data (indexes into a []string)
    * The Template parser will refuse to accept any template action that is not a case-sensitive reference to the CSV header row.
3. executes the template on each row of CSV data

# example:
given a simple template like the following:
```
Hello, {{ RecipientName }},

Welcome to {{ GroupName }}.

Thank you,
{{ SenderName }}
```
with a .csv document like the following
```
SenderName,RecipientName,GroupName
sender@example.com,recipient@example.com,group name
```
will generate
```
Hello, recipient@example.com,

Welcome to group name.

Thank you,
sender@example.com
```

# use:
```go
package main

func main() {
    // parse args for csv and template files
    f, err := os.ReadFile(csvpath)
    if err != nil {
        panic("error reading "+csvpath+": "+err.Error())
    }
    csv, err := template.NewCSV(csvpath, "text/csv", f)
    if err != nil {
        panic("error parsing "+csvpath+": "+err.Error())
    }
    f, err = os.ReadFile(templatepath)
    if err != nil {
        panic("error reading "+templatepath+": "+err.Error())
    }
    tmpl, err := template.NewTemplate(templatepath, "text/plain", f)
    if err != nil {
        panic("error parsing "+templatepath+": "+err.Error())
    }
    parsed, err := tmpl.ExecuteAll()
    if err != nil {
        panic("error executing template on csv: "+err.Error())
    }
    // work with parsed, write files, etc
}
```
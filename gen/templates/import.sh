terraform import ndfc_{{snakeCase .Name}}.example "{{$first := true}}{{range .Attributes}}{{if or .Id .Reference}}{{if not $first}}:{{end}}{{$first = false}}{{.Example}}{{end}}{{end}}"

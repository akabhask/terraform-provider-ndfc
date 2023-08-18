data "ndfc_{{snakeCase .Name}}" "example" {
	{{- range  .Attributes}}
	{{- if and (not .ExcludeTest) (not .TfOnly) (not .Value)}}
	{{- if or (eq .Type "List") (eq .Type "Set")}}
	{{.TfName}} = [{
		{{- range  .Attributes}}
		{{- if and (not .ExcludeTest) (not .TfOnly) (not .Value)}}
		{{- if or (eq .Type "List") (eq .Type "Set")}}
		{{.TfName}} = [{
			{{- range  .Attributes}}
			{{- if and (not .ExcludeTest) (not .TfOnly) (not .Value)}}
			{{- if or (eq .Type "List") (eq .Type "Set")}}
			{{.TfName}} = [{
				{{- range  .Attributes}}
				{{- if and (not .ExcludeTest) (not .TfOnly) (not .Value) (or .Id .Reference)}}
				{{.TfName}} = {{if eq .Type "String"}}"{{.Example}}"{{else if eq .Type "ListString"}}["{{.Example}}"]{{else}}{{.Example}}{{end}}
				{{- end}}
				{{- end}}
			}]
			{{- else if or .Id .Reference}}
			{{.TfName}} = {{if eq .Type "String"}}"{{.Example}}"{{else if eq .Type "ListString"}}["{{.Example}}"]{{else}}{{.Example}}{{end}}
			{{- end}}
			{{- end}}
			{{- end}}
		}]
		{{- else if or .Id .Reference}}
		{{.TfName}} = {{if eq .Type "String"}}"{{.Example}}"{{else if eq .Type "ListString"}}["{{.Example}}"]{{else}}{{.Example}}{{end}}
		{{- end}}
		{{- end}}
		{{- end}}
	}]
	{{- else if or .Id .Reference}}
	{{.TfName}} = {{if eq .Type "String"}}"{{.Example}}"{{else if eq .Type "ListString"}}["{{.Example}}"]{{else}}{{.Example}}{{end}}
	{{- end}}
	{{- end}}
	{{- end}}
}

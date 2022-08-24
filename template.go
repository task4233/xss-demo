package xssdemo

import (
	"bytes"
	"text/template"
)

func RenderTemplate(tmpl string, data interface{}) (string, error) {
	tpl, err := template.New("detail").Parse(tmpl)
	if err != nil {
		return "", err
	}

	writer := &bytes.Buffer{}
	err = tpl.Execute(writer, data)
	if err != nil {
		return "", err
	}

	return writer.String(), nil
}

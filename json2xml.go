package main

import (
	"encoding/json"
	"encoding/xml"
	"iter"
	"strconv"
)

const (
	rootName = iota
	nsJsonName
	nsArrayName
	nsObjectName
	valueName
	typeName
	lengthName
	itemName
	indexName
	fieldsName
	fieldName
	nameName
)

var names = [...]xml.Name{
	rootName:     {Space: "json", Local: "root"},
	nsJsonName:   {Space: "xmlns", Local: "json"},
	nsArrayName:  {Space: "xmlns", Local: "array"},
	nsObjectName: {Space: "xmlns", Local: "object"},
	valueName:    {Space: "json", Local: "value"},
	typeName:     {Space: "json", Local: "type"},
	lengthName:   {Space: "array", Local: "length"},
	itemName:     {Space: "array", Local: "item"},
	indexName:    {Space: "array", Local: "item-index"},
	fieldsName:   {Space: "object", Local: "fields"},
	fieldName:    {Space: "object", Local: "field"},
	nameName:     {Space: "object", Local: "field-name"},
}

type (
	tokenizer interface {
		tokens() (xml.StartElement, any)
	}

	root struct {
		tokenizer
		arrs, objs bool
	}

	value struct {
		parent tokenizer
		token  json.Token
	}

	array struct {
		parent tokenizer
		items  []*item
	}

	item struct {
		index int
		tokenizer
	}

	object struct {
		parent tokenizer
		fields []*field
	}

	field struct {
		name string
		tokenizer
	}

	typ string
)

const (
	nullType   typ = "null"
	boolType   typ = "bool"
	numberType typ = "number"
	stringType typ = "string"
	arrayType  typ = "array"
	objectType typ = "object"
)

func addType(st *xml.StartElement, t typ) {
	if flags.types {
		st.Attr = append(st.Attr, xml.Attr{Name: names[typeName], Value: string(t)})
	}
}

func (rt *root) tokens() (start xml.StartElement, rest any) {
	return rt.start(), rt.tokenizer
}

func (rt *root) start() (st xml.StartElement) {
	st.Name = names[rootName]
	if !flags.ns {
		return
	}
	st.Attr = append(st.Attr, xml.Attr{Name: names[nsJsonName], Value: "json-values"})
	if rt.arrs {
		st.Attr = append(st.Attr, xml.Attr{Name: names[nsArrayName], Value: "array-properties"})
	}
	if rt.objs {
		st.Attr = append(st.Attr, xml.Attr{Name: names[nsObjectName], Value: "object-properties"})
	}
	return
}

func (v *value) tokens() (st xml.StartElement, rest any) {
	st, val := v.start()
	if flags.attrValues {
		st.Attr = append(st.Attr, xml.Attr{Name: names[valueName], Value: val})
	}
	if !flags.attrValues {
		rest = xml.CharData(val)
	}
	return
}

func (v *value) start() (start xml.StartElement, val string) {
	t, val := v.val()

	switch p := v.parent.(type) {
	case nil:
		start.Name = names[valueName]
	case *item:
		start = p.start()
	case *field:
		start = p.start()
	default:
		panic("invalid parent")
	}

	addType(&start, t)
	return
}

func (v *value) val() (typ, string) {
	switch v := v.token.(type) {
	case string:
		return stringType, v
	case json.Number:
		return numberType, v.String()
	case bool:
		if v {
			return boolType, "true"
		}
		return boolType, "false"
	case nil:
		return nullType, "null"
	default:
		panic(v)
	}
}

func (v *value) attr() (a xml.Attr, ok bool) {
	fld, ok := v.parent.(*field)
	if ok = ok && validName(fld.name); ok {
		_, a.Value = v.val()
		a.Name.Local = fld.name
	}
	return
}

func (arr *array) tokens() (start xml.StartElement, rest any) {
	var seq iter.Seq[tokenizer] = func(yield func(tokenizer) bool) {
		for _, c := range arr.items {
			if !yield(c) {
				return
			}
		}
	}
	return arr.start(), seq
}

func (arr *array) start() (start xml.StartElement) {
	switch p := arr.parent.(type) {
	case nil:
		start.Name = names[valueName]
	case *item:
		start = p.start()
	case *field:
		start = p.start()
	default:
		panic("invalid parent")
	}

	addType(&start, arrayType)
	if flags.itemCount {
		start.Attr = append(start.Attr, xml.Attr{Name: names[lengthName], Value: strconv.Itoa(len(arr.items))})
	}
	return
}

func (i *item) start() xml.StartElement {
	st := xml.StartElement{Name: names[itemName]}
	if flags.indices {
		st.Attr = append(st.Attr, xml.Attr{Name: names[indexName], Value: strconv.Itoa(i.index)})
	}
	return st
}

func (ob *object) tokens() (st xml.StartElement, rest any) {
	st = ob.start()
	counts := map[string]int{}
	for _, f := range ob.fields {
		counts[f.name]++
	}
	for _, f := range ob.fields {
		if counts[f.name] == 1 {
			if attr, ok := f.attr(); ok {
				st.Attr = append(st.Attr, attr)
				delete(counts, f.name)
			}
		}
	}

	var seq iter.Seq[tokenizer] = func(yield func(tokenizer) bool) {
		for _, f := range ob.fields {
			if counts[f.name] > 0 && !yield(f) {
				return
			}
		}
	}
	return st, seq
}

func (ob *object) start() (start xml.StartElement) {
	switch p := ob.parent.(type) {
	case nil:
		start.Name = names[valueName]
	case *item:
		start = p.start()
	case *field:
		start = p.start()
	default:
		panic("invalid parent")
	}

	addType(&start, objectType)
	if flags.fieldCount {
		start.Attr = append(start.Attr, xml.Attr{Name: names[fieldsName], Value: strconv.Itoa(len(ob.fields))})
	}
	return
}

func (f *field) start() (st xml.StartElement) {
	if flags.fieldElems && validName(f.name) {
		st.Name.Local = f.name
	} else {
		st.Name = names[fieldName]
		st.Attr = append(st.Attr, xml.Attr{Name: names[nameName], Value: f.name})
	}
	return
}

func (f *field) attr() (_ xml.Attr, _ bool) {
	if flags.fieldAttrs {
		if v, ok := f.tokenizer.(*value); ok {
			return v.attr()
		}
	}
	return
}

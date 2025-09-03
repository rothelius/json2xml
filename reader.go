package main

import (
	"encoding/json"
	"fmt"
	"io"
)

func read(r io.Reader) (rt *root, err error) {
	rt = new(root)
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	var getObj func(tokenizer) (tokenizer, error)
	getObj = func(parent tokenizer) (_ tokenizer, err error) {
		tok, err := decoder.Token()
		if err != nil {
			return
		}

		switch t := tok.(type) {
		case bool, json.Number, string, nil:
			return &value{parent: parent, token: tok}, nil
		case json.Delim:
			switch t {
			case '[':
				rt.arrs = true
				arr := &array{parent: parent}
				i := 0
				for decoder.More() {
					it := &item{index: i}
					i++
					it.tokenizer, err = getObj(it)
					if err != nil {
						return nil, fmt.Errorf("error reading array item: %w", err)
					}
					arr.items = append(arr.items, it)
				}
				if t, _ := decoder.Token(); t != json.Delim(']') {
					panic("unmatched array")
				}
				return arr, nil
			case '{':
				rt.objs = true
				objt := &object{parent: parent}
				for decoder.More() {
					name, err := decoder.Token()
					if err != nil {
						return nil, fmt.Errorf("error reading object name: %w", err)
					}
					sname, ok := name.(string)
					if !ok {
						return nil, fmt.Errorf("object name was a %T, not a string", name)
					}
					f := &field{name: sname}

					f.tokenizer, err = getObj(f)
					if err != nil {
						return nil, fmt.Errorf("error reading object item: %w", err)
					}
					objt.fields = append(objt.fields, f)
				}
				if t, _ := decoder.Token(); t != json.Delim('}') {
					panic("unmatched object")
				}
				return objt, nil
			default:
				return nil, fmt.Errorf("invalid delim: %c", t)
			}
		default:
			return nil, fmt.Errorf("invalid token type: %T", tok)
		}
	}

	rt.tokenizer, err = getObj(nil)
	if err == io.EOF {
		err = nil
	}
	return
}

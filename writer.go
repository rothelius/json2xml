package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"iter"
	"os"
)

type tokenWriter struct {
	prevStart bool
	buf       bytes.Buffer
	enc       *xml.Encoder
}

func (tw *tokenWriter) flush() {
	tw.enc.Flush()
	io.Copy(os.Stdout, &tw.buf)
	tw.buf.Reset()
}

const score = "______________"

func nameSpaced(n *xml.Name) {
	if flags.ns && n.Space != "" {
		n.Local = score[:len(n.Space)+1] + n.Local
	}
	n.Space = ""
}

func spaceOut(s xml.StartElement) (S xml.StartElement) {
	S = s.Copy()
	nameSpaced(&S.Name)
	for i := range S.Attr {
		nameSpaced(&S.Attr[i].Name)
	}
	return
}

func (tw *tokenWriter) correctNS(st xml.StartElement) {
	bs := tw.buf.Bytes()
	//	fmt.Printf("Before: %s\n", bs)
	i := bytes.IndexByte(bs, '<') + 1
	if st.Name.Space != "" {
		_ = append(append(bs[:i], st.Name.Space...), ':')
		//	fmt.Printf("After fixing name %v: %s\n", st.Name, bs)
	}
	i = bytes.IndexByte(bs, ' ') - 1
	for ; len(st.Attr) > 0; st.Attr = st.Attr[1:] {
		bs = bs[i+2:]
		if spc := st.Attr[0].Name.Space; spc != "" {
			_ = append(append(bs[:0], spc...), ':')
		}
		//	fmt.Printf("After fixing attr %v: %s\n", st.Attr[0].Name, bs)
		i = bytes.IndexByte(bs, '"')
		bs = bs[i+1:]
		i = bytes.IndexByte(bs, '"')

	}
}

func (tw *tokenWriter) writeToken(t xml.Token) (err error) {
	switch t := t.(type) {
	case xml.StartElement:
		tw.prevStart = true
		tw.flush()
		if err = tw.enc.EncodeToken(spaceOut(t)); err != nil {
			return fmt.Errorf("failed writing start token %v: %w", t.Name, err)
		}
		tw.enc.Flush()
		if flags.ns {
			tw.correctNS(t)
		}
		return nil
	case xml.EndElement:
		onm := t.Name
		nameSpaced(&t.Name)
		if tw.prevStart {
			bs := tw.buf.Bytes()
			io.Copy(os.Stdout, bytes.NewReader(bs[:len(bs)-1]))
			os.Stdout.Write([]byte("/>"))
			tw.buf.Reset()
			err = tw.enc.EncodeToken(t)
			tw.enc.Flush()
			tw.buf.Reset()
		} else {
			tw.flush()
			err = tw.enc.EncodeToken(t)
			if flags.ns && onm.Space != "" && err == nil {
				tw.enc.Flush()
				bs := tw.buf.Bytes()
				_ = append(append(bs[:bytes.IndexByte(bs, '/')+1], onm.Space...), ':')
			}
		}
		if err != nil {
			return fmt.Errorf("failed writing end token %v: %w", t.Name, err)
		}
	default:
		if err = tw.enc.EncodeToken(t); err != nil {
			return fmt.Errorf("failed writing content %T: %w", t, err)
		}
		tw.flush()
	}
	tw.prevStart = false
	return
}

func visitTokenser(ts tokenizer, yield func(xml.Token) bool) bool {
	st, rest := ts.tokens()
	if !yield(st) {
		return false
	}
	switch t := rest.(type) {
	case nil:
	case tokenizer:
		if !visitTokenser(t, yield) {
			return false
		}
	case iter.Seq[tokenizer]:
		for t := range t {
			if !visitTokenser(t, yield) {
				return false
			}
		}
	case iter.Seq[xml.Token]:
		for t := range t {
			if !yield(t) {
				return false
			}
		}
	case xml.Token:
		if !yield(t) {
			return false
		}
	default:
		panic(t)
	}

	return yield(st.End())
}

func tokenizerToSeq(ts tokenizer) iter.Seq[xml.Token] {
	return func(yield func(xml.Token) bool) {
		visitTokenser(ts, yield)
	}
}

func (tw *tokenWriter) writeTokenser(t tokenizer) error {
	for t := range tokenizerToSeq(t) {
		if err := tw.writeToken(t); err != nil {
			return fmt.Errorf("failed writing token of type %T: %w", t, err)
		}
	}
	return nil
}

func newTokenWriter() *tokenWriter {
	var tw tokenWriter
	tw.enc = xml.NewEncoder(&tw.buf)
	if flags.pretty {
		tw.enc.Indent("", "\t")
	}
	return &tw
}

package main

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"os"
)

var flags struct {
	attrValues,
	fieldElems,
	fieldAttrs,
	itemCount,
	fieldCount,
	indices,
	ns,
	pretty,
	types bool
}

func main() {
	flag.BoolVar(&flags.attrValues, "attr-values", false, "Force all values to be attributes in suitable elements.")
	flag.BoolVar(&flags.fieldElems, "field-elems", true, "Use field names as element names when possible.")
	flag.BoolVar(&flags.fieldAttrs, "field-attrs", true, "Use field names as attribute names when possible.")
	flag.BoolVar(&flags.itemCount, "item-count", true, "Include attribute containing array length.")
	flag.BoolVar(&flags.fieldCount, "field-count", true, "Include attribute containing object field count.")
	flag.BoolVar(&flags.indices, "indices", true, "Include attribute containing item index.")
	flag.BoolVar(&flags.ns, "ns", false, "Qualify internal entities with XML namespaces.")
	flag.BoolVar(&flags.pretty, "pretty", true, "Pretty-print output.")
	flag.BoolVar(&flags.types, "types", false, "Include type information.")
	flag.Parse()

	rt, err := read(os.Stdin)
	if err != nil {
		log.Fatalf("Failed parsing input: %v", err)
	}
	tw := newTokenWriter()
	err = tw.writeTokenser(rt)
	if err == nil {
		tw.flush()
		fmt.Println()
	}

	if err = cmp.Or(err, tw.enc.Close()); err != nil {
		log.Fatal(err)
	}
}

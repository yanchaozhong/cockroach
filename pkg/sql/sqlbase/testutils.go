// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package sqlbase

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"
	"unicode"

	"golang.org/x/net/context"

	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/keys"
	"github.com/cockroachdb/cockroach/pkg/sql/parser"
	"github.com/cockroachdb/cockroach/pkg/util/duration"
	"github.com/cockroachdb/cockroach/pkg/util/ipaddr"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/uuid"
)

// This file contains utility functions for tests (in other packages).

// GetTableDescriptor retrieves a table descriptor directly from the KV layer.
func GetTableDescriptor(kvDB *client.DB, database string, table string) *TableDescriptor {
	dbNameKey := MakeNameMetadataKey(keys.RootNamespaceID, database)
	gr, err := kvDB.Get(context.TODO(), dbNameKey)
	if err != nil {
		panic(err)
	}
	if !gr.Exists() {
		panic("database missing")
	}
	dbDescID := ID(gr.ValueInt())

	tableNameKey := MakeNameMetadataKey(dbDescID, table)
	gr, err = kvDB.Get(context.TODO(), tableNameKey)
	if err != nil {
		panic(err)
	}
	if !gr.Exists() {
		panic("table missing")
	}

	descKey := MakeDescMetadataKey(ID(gr.ValueInt()))
	desc := &Descriptor{}
	if err := kvDB.GetProto(context.TODO(), descKey, desc); err != nil || (*desc == Descriptor{}) {
		log.Fatalf(context.TODO(), "proto with id %d missing. err: %v", gr.ValueInt(), err)
	}
	return desc.GetTable()
}

// RandDatum generates a random Datum of the given type.
// If null is true, the datum can be DNull.
// Note that if typ.SemanticType is ColumnType_NULL, the datum will always be DNull,
// regardless of the null flag.
func RandDatum(rng *rand.Rand, typ ColumnType, null bool) parser.Datum {
	if null && rng.Intn(10) == 0 {
		return parser.DNull
	}
	switch typ.SemanticType {
	case ColumnType_BOOL:
		return parser.MakeDBool(rng.Intn(2) == 1)
	case ColumnType_INT:
		return parser.NewDInt(parser.DInt(rng.Int63()))
	case ColumnType_FLOAT:
		return parser.NewDFloat(parser.DFloat(rng.NormFloat64()))
	case ColumnType_DECIMAL:
		d := &parser.DDecimal{}
		d.Decimal.SetExponent(int32(rng.Intn(40) - 20))
		d.Decimal.SetCoefficient(rng.Int63())
		return d
	case ColumnType_DATE:
		return parser.NewDDate(parser.DDate(rng.Intn(10000)))
	case ColumnType_TIMESTAMP:
		return &parser.DTimestamp{Time: time.Unix(rng.Int63n(1000000), rng.Int63n(1000000))}
	case ColumnType_INTERVAL:
		sign := 1 - rng.Int63n(2)*2
		return &parser.DInterval{Duration: duration.Duration{
			Months: sign * rng.Int63n(1000),
			Days:   sign * rng.Int63n(1000),
			Nanos:  sign * rng.Int63n(25*3600*int64(1000000000)),
		}}
	case ColumnType_UUID:
		return parser.NewDUuid(parser.DUuid{UUID: uuid.MakeV4()})
	case ColumnType_INET:
		ipAddr := ipaddr.RandIPAddr(rng)
		return parser.NewDIPAddr(parser.DIPAddr{IPAddr: ipAddr})
	case ColumnType_STRING:
		// Generate a random ASCII string.
		p := make([]byte, rng.Intn(10))
		for i := range p {
			p[i] = byte(1 + rng.Intn(127))
		}
		return parser.NewDString(string(p))
	case ColumnType_BYTES:
		p := make([]byte, rng.Intn(10))
		_, _ = rng.Read(p)
		return parser.NewDBytes(parser.DBytes(p))
	case ColumnType_TIMESTAMPTZ:
		return &parser.DTimestampTZ{Time: time.Unix(rng.Int63n(1000000), rng.Int63n(1000000))}
	case ColumnType_COLLATEDSTRING:
		if typ.Locale == nil {
			panic("locale is required for COLLATEDSTRING")
		}
		// Generate a random Unicode string.
		var buf bytes.Buffer
		n := rng.Intn(10)
		for i := 0; i < n; i++ {
			var r rune
			for {
				r = rune(rng.Intn(unicode.MaxRune + 1))
				if !unicode.Is(unicode.C, r) {
					break
				}
			}
			buf.WriteRune(r)
		}
		return parser.NewDCollatedString(buf.String(), *typ.Locale, &parser.CollationEnvironment{})
	case ColumnType_NAME:
		// Generate a random ASCII string.
		p := make([]byte, rng.Intn(10))
		for i := range p {
			p[i] = byte(1 + rng.Intn(127))
		}
		return parser.NewDName(string(p))
	case ColumnType_OID:
		return parser.NewDOid(parser.DInt(rng.Int63()))
	case ColumnType_NULL:
		return parser.DNull
	case ColumnType_ARRAY:
		// TODO(justin)
		return parser.DNull
	case ColumnType_INT2VECTOR:
		return parser.DNull
	default:
		panic(fmt.Sprintf("invalid type %s", typ.String()))
	}
}

var (
	columnSemanticTypes []ColumnType_SemanticType
	collationLocales    = [...]string{"da", "de", "en"}
)

func init() {
	for k := range ColumnType_SemanticType_name {
		columnSemanticTypes = append(columnSemanticTypes, ColumnType_SemanticType(k))
	}
}

// RandCollationLocale returns a random element of collationLocales.
func RandCollationLocale(rng *rand.Rand) *string {
	return &collationLocales[rng.Intn(len(collationLocales))]
}

// RandColumnType returns a random ColumnType value.
func RandColumnType(rng *rand.Rand) ColumnType {
	typ := ColumnType{SemanticType: columnSemanticTypes[rng.Intn(len(columnSemanticTypes))]}
	if typ.SemanticType == ColumnType_COLLATEDSTRING {
		typ.Locale = RandCollationLocale(rng)
	}
	if typ.SemanticType == ColumnType_ARRAY {
		typ.ArrayContents = &columnSemanticTypes[rng.Intn(len(columnSemanticTypes))]
		if *typ.ArrayContents == ColumnType_COLLATEDSTRING {
			// TODO(justin): change this when collated arrays are supported.
			s := ColumnType_STRING
			typ.ArrayContents = &s
		}
	}
	return typ
}

// RandColumnTypes returns a slice of numCols random ColumnType value.
func RandColumnTypes(rng *rand.Rand, numCols int) []ColumnType {
	types := make([]ColumnType, numCols)
	for i := range types {
		types[i] = RandColumnType(rng)
	}
	return types
}

// RandDatumEncoding returns a random DatumEncoding value.
func RandDatumEncoding(rng *rand.Rand) DatumEncoding {
	return DatumEncoding(rng.Intn(len(DatumEncoding_value)))
}

// RandEncDatum generates a random EncDatum (of a random type).
func RandEncDatum(rng *rand.Rand) EncDatum {
	typ := RandColumnType(rng)
	datum := RandDatum(rng, typ, true)
	return DatumToEncDatum(typ, datum)
}

// RandEncDatumSlice generates a slice of random EncDatum values of the same random
// type.
func RandEncDatumSlice(rng *rand.Rand, numVals int) []EncDatum {
	typ := RandColumnType(rng)
	vals := make([]EncDatum, numVals)
	for i := range vals {
		vals[i] = DatumToEncDatum(typ, RandDatum(rng, typ, true))
	}
	return vals
}

// RandEncDatumSlices generates EncDatum slices, each slice with values of the same
// random type.
func RandEncDatumSlices(rng *rand.Rand, numSets, numValsPerSet int) [][]EncDatum {
	vals := make([][]EncDatum, numSets)
	for i := range vals {
		vals[i] = RandEncDatumSlice(rng, numValsPerSet)
	}
	return vals
}

// RandEncDatumRowOfTypes generates a slice of random EncDatum values for the
// corresponding type in types.
func RandEncDatumRowOfTypes(rng *rand.Rand, types []ColumnType) EncDatumRow {
	vals := make([]EncDatum, len(types))
	for i, typ := range types {
		vals[i] = DatumToEncDatum(typ, RandDatum(rng, typ, true))
	}
	return vals
}

// RandEncDatumRows generates EncDatumRows where all rows follow the same random
// []ColumnType structure.
func RandEncDatumRows(rng *rand.Rand, numRows, numCols int) EncDatumRows {
	return RandEncDatumRowsOfTypes(rng, numRows, RandColumnTypes(rng, numCols))
}

// RandEncDatumRowsOfTypes generates EncDatumRows, each row with values of the
// corresponding type in types.
func RandEncDatumRowsOfTypes(rng *rand.Rand, numRows int, types []ColumnType) EncDatumRows {
	vals := make(EncDatumRows, numRows)
	for i := range vals {
		vals[i] = RandEncDatumRowOfTypes(rng, types)
	}
	return vals
}

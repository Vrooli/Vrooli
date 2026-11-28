package plans

import (
	"reflect"
	"testing"

	"plan-manager/internal/planmodel"

	"github.com/stretchr/testify/require"
)

// TestContentHashMatchesPlanFieldClassification makes the classification table
// executable. A future top-level field cannot silently become hash-visible (or
// hash-invisible) without changing its declared class and this assertion.
func TestContentHashMatchesPlanFieldClassification(t *testing.T) {
	base := contentHash(Plan{})
	typ := reflect.TypeOf(Plan{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		mutated := reflect.New(typ).Elem()
		populateFieldValue(mutated.Field(i))
		got := contentHash(mutated.Interface().(Plan))
		if planmodel.PlanFieldClasses[field.Name] == planmodel.FieldClassAuthored {
			require.NotEqualf(t, base, got, "authored Plan.%s must affect content hash", field.Name)
		} else {
			require.Equalf(t, base, got, "non-authored Plan.%s must not affect content hash", field.Name)
		}
	}
}

func populateFieldValue(value reflect.Value) {
	switch value.Kind() {
	case reflect.String:
		value.SetString("changed")
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(1)
	case reflect.Float32, reflect.Float64:
		value.SetFloat(1)
	case reflect.Slice:
		item := reflect.New(value.Type().Elem()).Elem()
		populateFieldValue(item)
		value.Set(reflect.Append(value, item))
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		populateFieldValue(value.Elem())
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if value.Type().Field(i).PkgPath == "" {
				populateFieldValue(value.Field(i))
			}
		}
	default:
		panic("unsupported Plan field kind: " + value.Kind().String())
	}
}

package rehearsals

import (
	"encoding/json"
	"testing"
)

func TestDebug19880(t *testing.T) { b:=readFixture(t,"public-workflows/19880.json");var d struct{Nodes []struct{Name string `json:"name"`;Type string `json:"type"`;Parameters struct{JS string `json:"jsCode"`} `json:"parameters"`} `json:"nodes"`};if err:=json.Unmarshal([]byte(b),&d);err!=nil{t.Fatal(err)};for _,n:=range d.Nodes{if n.Type=="n8n-nodes-base.code"{_,err:=parseSafeCode(n.Parameters.JS);t.Logf("%s: %v",n.Name,err)}} }

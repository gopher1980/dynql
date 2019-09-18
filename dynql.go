package dynql

import (
	"encoding/json"
	"github.com/yalp/jsonpath"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"sync"
)

type Handler func(name string, i interface{}, r *http.Request, payload interface{}) interface{}

type DQL struct {
	handlers   map[string]Handler
	parameters map[string]interface{}
}

type ParamQuery struct {
	Method    string `json:"method"`
	Input interface{}
	Output     map[string]string
}

func NewDQL() *DQL {
	return &DQL{handlers: make(map[string]Handler), parameters: make(map[string]interface{})}
}

func (dql DQL) Put(name string, handler Handler, param interface{}) {
	dql.handlers[name] = handler
	dql.parameters[name] = param
}

func (dql DQL) Get(name string) Handler {
	return dql.handlers[name]
}

func (dql DQL) Run(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	body, err := ioutil.ReadAll(r.Body)
	mapQuery := make(map[string]ParamQuery)
	mapQueryReturn := make(map[string]interface{})
	if err != nil {
		_ = json.NewEncoder(w).Encode(err)
	}
	_ = json.Unmarshal(body, &mapQuery)

	keys  := []string{}
	for k, _ := range mapQuery {
		keys = append(keys,k)
	}
	sort.Strings(keys)

	var m sync.Mutex
	var prevElement interface{}
	for _, k := range keys {
		func () {
			m.Lock()
			defer func() {m.Unlock()}()

			paramQuery := mapQuery[k]
			realMethod := paramQuery.Method
			if dql.handlers[paramQuery.Method] == nil {
				paramQuery.Method = "default"
			}

			paramByte, _ := json.Marshal(paramQuery.Input)
			param := reflect.New(reflect.TypeOf(dql.parameters[paramQuery.Method])).Interface()
			json.Unmarshal(paramByte, param)
			elem := dql.handlers[paramQuery.Method](realMethod, param, r, prevElement)
			prevElement := elem
			if paramQuery.Output == nil {
				mapQueryReturn [k] = elem
				return
			}

			result := make(map[string]interface{})
			for k, v := range paramQuery.Output {
				var payload interface{}
				var sample []byte
				sample, _ = json.Marshal(elem)
				_ = json.Unmarshal(sample, &payload)
				result[k], err = jsonpath.Read(payload, v)
				if err != nil {
					result[k] = err
				}
			}
			mapQueryReturn [k] = result
		}()
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		q = "$"
	}

	var payload interface{}
	var sample []byte
	sample, _ = json.Marshal(mapQueryReturn)

	_ = json.Unmarshal(sample, &payload)

	result, err := jsonpath.Read(payload, q)
	if err != nil {
		json.NewEncoder(w).Encode(err)
		return
	}

	json.NewEncoder(w).Encode(result)

}

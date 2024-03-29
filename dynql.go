package dynql

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"sync"

	"github.com/yalp/jsonpath"
)

type Handler func(name string, i interface{}, r *http.Request, payload interface{}, parent interface{}) interface{}

type DQL struct {
	handlers   map[string]Handler
	parameters map[string]interface{}
}

type ParamQuery struct {
	Method string `json:"method"`
	Hidden bool
	Name   string
	Input  interface{}
	Output map[string]string
	Fork   map[string]ParamQuery
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

func (dql DQL) run(pMapQuery *map[string]ParamQuery, r *http.Request, prevElement interface{}, parent interface{}) (interface{}, error) {
	keys := []string{}
	mapQuery := *pMapQuery
	mapQueryReturn := make(map[string]interface{})

	for k, _ := range mapQuery {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var m sync.Mutex
	for _, k := range keys {
		func() {
			m.Lock()
			defer func() { m.Unlock() }()

			paramQuery := mapQuery[k]
			realMethod := paramQuery.Method
			if dql.handlers[paramQuery.Method] == nil {
				paramQuery.Method = "default"
			}

			paramByte, _ := json.Marshal(paramQuery.Input)
			param := reflect.New(reflect.TypeOf(dql.parameters[paramQuery.Method])).Interface()
			json.Unmarshal(paramByte, param)
			elem := dql.handlers[paramQuery.Method](realMethod, param, r, prevElement, parent)
			prevElement = elem

			if paramQuery.Fork != nil {
				item := make(map[string]interface{})
				for k, v := range elem.(map[string]interface{}) {
					item[k] = v
				}
				result, err := dql.run(&paramQuery.Fork, r, prevElement, prevElement)
				if err != nil {
					item["error"] = err
				} else {
					if result != nil {
						for k, v := range result.(map[string]interface{}) {
							item[k] = v
						}
					}
				}

				elem = item
				prevElement = elem
			}

			keyReport := k
			if paramQuery.Name != "" {
				keyReport = paramQuery.Name
			}
			if paramQuery.Output == nil {
				if !paramQuery.Hidden {
					mapQueryReturn[keyReport] = elem
				}
				return
			}

			if !paramQuery.Hidden {
				result := make(map[string]interface{})
				for k, v := range paramQuery.Output {
					var payload interface{}
					var sample []byte
					sample, _ = json.Marshal(elem)
					_ = json.Unmarshal(sample, &payload)
					var err error
					result[k], err = jsonpath.Read(payload, v)
					if err != nil {
						result[k] = err
					}
				}
				mapQueryReturn[keyReport] = result
			}
		}()
	}

	var payload interface{}
	var sample []byte
	sample, _ = json.Marshal(mapQueryReturn)
	_ = json.Unmarshal(sample, &payload)
	return payload, nil

}

// Run execute query
func (dql DQL) Run(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	body, err := ioutil.ReadAll(r.Body)
	mapQuery := make(map[string]ParamQuery)
	if err != nil {
		_ = json.NewEncoder(w).Encode(err)
	}
	_ = json.Unmarshal(body, &mapQuery)
	payload, err := dql.run(&mapQuery, r, nil, nil)

	q := r.URL.Query().Get("q")
	if q == "" {
		q = "$"
	}
	result, err := jsonpath.Read(payload, q)

	if err != nil {
		json.NewEncoder(w).Encode(err)
		return
	}
	json.NewEncoder(w).Encode(result)
}

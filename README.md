# dynql
 

DynQL is has one enpoint for query and tranform the Entities.

Example 
POST -> 127.0.0.1:9090/dql

Input: 
```json
{
    "query1": {
        "method": "demo",
        "input": {
            "name": "Pedro Martinez",
            "age": 59,
            "position": {
                "name": "manager"
            }
        },
        "output": {
            "first_name": "$.name",
            "age": "$.age",
            "level": "$.position.name"
        }
    },
    "query2": {
        "method": "demo",
        "input": {
            "name": "Marcela Perez",
            "age": 31
        }
    }
}
```

Output
```json
{
    "query1": {
        "age": 59,
        "first_name": "Pedro Martinez",
        "level": "manager"
    },
    "query2": {
        "age": 31,
        "name": "Marcela Perez",
        "position": {
            "name": ""
        }
    }
}
```


```go
package main

import (
	"github.com/gopher1980/dynql"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)
type Position struct {
	Name string `json:"name"`
}
type Persona struct {
	Name     string   `json:"name"`
	Age      int      `json:"age"`
	Position Position `json:"position"`
}
func demo(name string, ptr interface{}, r *http.Request,payload interface{}, parent interface{}) interface{} {
	p := ptr.(*Persona)
	return p

}
func main() {
	dql := dynql.NewDQL()
	dql.Put("demo", demo, Persona{})
	r := mux.NewRouter()
	r.HandleFunc("/dql", dql.Run).Methods(http.MethodPost)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":9090", nil))
}
```


Example without strictic struct:
```go
package main

import (
	"github.com/gopher1980/dynql"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func demo(name string, ptr interface{}, r *http.Request,payload interface{}, parent interface{}) interface{} {
	return ptr

}
func main() {
	dql := dynql.NewDQL()
	dql.Put("demo", demo, make(map[string]interface{}))
	r := mux.NewRouter()
	r.HandleFunc("/dql", dql.Run).Methods(http.MethodPost)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":9090", nil))
}

```





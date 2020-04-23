package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// PROBLEM DESCRIPTION:
// Your goal here is to design an API that allows for hero tracking, much like the Vue problem
// You are to implement an API (for which the skeleton already exists) that has the following capabilities
// - Get      : return a JSON representation of the hero with the name supplied
// - Make     : create a superhero according to the JSON body supplied
// - Calamity : a calamity of the supplied level requires heroes with an equivalent combined powerlevel to address it.
//              Takes a calamity with powerlevel and at least 1 hero. On success return a 200 with json response indicating the calamity has been resolved.
//              Otherwise return a response indicating that the heroes were not up to the task. Addressing a calamity adds 1 point of exhaustion.
// - Rest     : recover 1 point of exhaustion
// - Retire   : retire a superhero, someone may take up his name for the future passing on the title
// - Kill     : a superhero has passed away, his name may not be taken up again.

// On success all endpoints should return a status code 200.

// If a hero reaches an exhaustion level of maxExhaustion then they die.

// You are free to decide what your API endpoints should be called and what shape they should take. You can modify any code in this file however you'd like.

// NOTE: you may want to install postman or another request generating software of your choosing to make testing easier. (api is running on localhost port 8081)

// NOTE the second: the API is receiving asynchronous requests to manage our super friends. As such, your hero access should be thread-safe.
// Even if the operations are extremely lightweight we want to make our application scalable.
// Your multithreaded protection should allow the API to still be performant even if these requests took a reasonably long time. This means that
// a global lock/mutex that makes the API only handle 1 request at a time is not an acceptable solution to this problem

// NOTE the third: There are many ways to make whatever package-level tracking you implement thread-safe, feel free to change anything about this file (without changing the functionality of the program) to do so.
// i.e. add package-level maps, add functions that take the hero struct as reference, modify the hero struct, creating access control paradigms
// I highly recommend looking into channels, mutexes, and other golang memory management patterns and pick whatever you're most comfortable with.
// For mad props: a timeout on the memory management process which returns a resource not available.

// Bonus: If you're having fun (this is by no means necessary) you can make the calamity hold the heroes up for a time and delay their unlocking in a go-routine
// example:
// go func(h *hero) {
//     time.Sleep(calamityTime)
//     // release lock on hero
// }(heroPtr)

var maxExhaustion = 10

type hero struct {
	PowerLevel int    `json:"PowerLevel"`
	Exhaustion int    `json:"Exhaustion"`
	Name       string `json:"Name"`
	Alive      bool
}

type calamity struct {
	PowerLevel int      `json:"PowerLevel"`
	Heroes     []string `json:"Heroes"`
}

type mapOfHeroes struct {
	sync.RWMutex
	actualMap map[string]hero
}

func (heroes *mapOfHeroes) load(key string) (value hero, ok bool) {
	heroes.RLock()
	result, ok := heroes.actualMap[key]
	heroes.RUnlock()
	return result, ok
}

func (heroes *mapOfHeroes) delete(key string) {
	heroes.Lock()
	delete(heroes.actualMap, key)
	heroes.Unlock()
}

func (heroes *mapOfHeroes) store(key string, value hero) {
	heroes.Lock()
	heroes.actualMap[key] = value
	heroes.Unlock()
}

var heroes mapOfHeroes

func handleCalamity(w http.ResponseWriter, r *http.Request) {
	content, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		http.Error(w, readErr.Error(), http.StatusInternalServerError)
		return
	}
	var calamity calamity
	unmarshalErr := json.Unmarshal(content, &calamity)
	if unmarshalErr != nil {
		http.Error(w, unmarshalErr.Error(), http.StatusInternalServerError)
		return
	}
	if len(calamity.Heroes) < 1 {
		http.Error(w, "Must designate one or more heroes to address the calamity", http.StatusInternalServerError)
		return
	}

	var heroesForCalamity []hero
	var totalPowerLevel int
	for _, heroName := range calamity.Heroes {
		var hero hero
		var heroExists bool
		if hero, heroExists = heroes.load(heroName); !heroExists {
			http.Error(w, "Hero with name "+heroName+" does not exist", http.StatusNotFound)
			return
		}
		if hero.Alive == false {
			http.Error(w, "Hero with name "+heroName+" is dead and can no longer fight", http.StatusNotFound)
			return
		}
		totalPowerLevel += hero.PowerLevel
		hero.Exhaustion++
		if hero.Exhaustion == maxExhaustion {
			hero.Alive = false
		}
		heroesForCalamity = append(heroesForCalamity, hero)
	}

	if calamity.PowerLevel == totalPowerLevel {
		for _, hero := range heroesForCalamity {
			heroes.store(hero.Name, hero)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Provided heros cannot tackle this calamity", http.StatusBadRequest)
	return
}

func heroKill(w http.ResponseWriter, r *http.Request) {
	var name string
	var ok bool
	if name, ok = mux.Vars(r)["name"]; !ok {
		http.Error(w, "A name must be provided", http.StatusInternalServerError)
		return
	}
	var hero hero
	if hero, ok = heroes.load(name); !ok {
		http.Error(w, "Hero with name "+name+" does not exist", http.StatusNotFound)
		return
	}
	hero.Alive = false
	heroes.store(name, hero)
	w.WriteHeader(http.StatusOK)
}

func heroRetire(w http.ResponseWriter, r *http.Request) {
	var name string
	var ok bool
	if name, ok = mux.Vars(r)["name"]; !ok {
		http.Error(w, "A name must be provided", http.StatusInternalServerError)
		return
	}
	var hero hero
	if hero, ok = heroes.load(name); !ok {
		http.Error(w, "Hero with name "+name+" does not exist", http.StatusNotFound)
		return
	}
	if hero.Alive != true {
		http.Error(w, "Hero with name "+name+" has been killed, and thus cannot retire", http.StatusNotFound)
		return
	}
	heroes.delete(name)
	w.WriteHeader(http.StatusOK)
}

func heroRest(w http.ResponseWriter, r *http.Request) {
	var name string
	var ok bool
	if name, ok = mux.Vars(r)["name"]; !ok {
		http.Error(w, "A name must be provided", http.StatusInternalServerError)
		return
	}
	var hero hero
	if hero, ok = heroes.load(name); !ok {
		http.Error(w, "Hero with name "+name+" does not exist", http.StatusNotFound)
		return
	}
	if hero.Exhaustion == 0 {
		http.Error(w, "Hero with name "+name+" does not need rest", http.StatusBadRequest)
		return
	}
	hero.Exhaustion--
	heroes.store(name, hero)
	w.WriteHeader(http.StatusOK)
}

func heroGet(w http.ResponseWriter, r *http.Request) {
	var name string
	var ok bool
	if name, ok = mux.Vars(r)["name"]; !ok {
		http.Error(w, "A name must be provided", http.StatusNotFound)
		return
	}
	var hero hero
	if hero, ok = heroes.load(name); !ok {
		http.Error(w, "Hero with name "+name+" does not exist", http.StatusNotFound)
		return
	}

	js, err := json.Marshal(hero)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}

func heroMake(w http.ResponseWriter, r *http.Request) {
	content, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		http.Error(w, readErr.Error(), http.StatusInternalServerError)
		return
	}
	var hero hero
	unmarshalErr := json.Unmarshal(content, &hero)
	if unmarshalErr != nil {
		http.Error(w, unmarshalErr.Error(), http.StatusInternalServerError)
		return
	}
	if hero, heroExists := heroes.load(hero.Name); heroExists {
		if hero.Alive {
			http.Error(w, "Hero with name "+hero.Name+" already exists", http.StatusConflict)
		} else {
			http.Error(w, "A hero named "+hero.Name+" once died valiantly in battle, and their name shall not be taken", http.StatusConflict)
		}
		return
	}
	hero.Alive = true
	heroes.store(hero.Name, hero)
	w.WriteHeader(http.StatusOK)
}

func linkRoutes(r *mux.Router) {
	r.HandleFunc("/hero", heroMake).Methods("POST")
	r.HandleFunc("/hero/{name}", heroGet).Methods("GET")
	r.HandleFunc("/hero/rest/{name}", heroRest).Methods("PATCH")
	r.HandleFunc("/hero/kill/{name}", heroKill).Methods("PATCH")
	r.HandleFunc("/hero/{name}", heroRetire).Methods("DELETE")

	r.HandleFunc("/calamity", handleCalamity).Methods("POST")
}

func main() {
	// Create map to hold the heroes data
	heroes = mapOfHeroes{
		actualMap: make(map[string]hero),
	}

	// Create a router
	router := mux.NewRouter()

	// Create a server to listen on port 8081
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", 8081),
		Handler: router,
	}

	// Link the supplied routes
	linkRoutes(router)

	// Wait for requests
	log.Fatal(server.ListenAndServe())
}

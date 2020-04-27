package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

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
var resourceNotAvailableErr = "Resource not available - Cannot access Hero Data"

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

var heroMapChannel chan map[string]hero

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

	mapOfHeroes := attemptToGetHeroData()
	if mapOfHeroes == nil {
		http.Error(w, resourceNotAvailableErr, http.StatusInternalServerError)
		return
	}

	var totalPowerLevel int
	var heroesForCalamity []hero
	var err error
	totalPowerLevel, heroesForCalamity, err = compileHeroData(calamity.Heroes, mapOfHeroes)
	if err != nil {
		heroMapChannel <- mapOfHeroes
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if calamity.PowerLevel > totalPowerLevel {
		heroMapChannel <- mapOfHeroes
		errMessage := "Powerlevel of calamity is higher than total powerlevel of all heroes. This calamity cannot be addressed."
		http.Error(w, errMessage, http.StatusBadRequest)
		return
	}

	for _, hero := range heroesForCalamity {
		hero.Exhaustion++
		if hero.Exhaustion == maxExhaustion {
			hero.Alive = false
		}
		mapOfHeroes[hero.Name] = hero
	}
	heroMapChannel <- mapOfHeroes
	w.WriteHeader(http.StatusOK)

}

// Loops through the list of names and pulls heroes off of the map. Each hero's powerlevel is added to the
// total powerlevel, and each hero is added to an array of heroes. At the end of the method, the total powerlevel
// and array of heroes are returned.
//
// If there is an issue obtaining the data, the error this function returns will not be nil.
func compileHeroData(names []string, mapOfHeroes map[string]hero) (int, []hero, error) {
	var heroesForCalamity []hero
	var totalPowerLevel int
	for _, heroName := range names {
		var hero hero
		var heroExists bool
		if hero, heroExists = mapOfHeroes[heroName]; !heroExists {
			return 0, nil, fmt.Errorf("Hero with name %q does not exist", heroName)
		}
		if !hero.Alive {
			return 0, nil, fmt.Errorf("Hero with name %q is dead and can no longer fight", heroName)
		}
		totalPowerLevel += hero.PowerLevel
		heroesForCalamity = append(heroesForCalamity, hero)
	}
	return totalPowerLevel, heroesForCalamity, nil
}

func heroKill(w http.ResponseWriter, r *http.Request) {
	var name string
	var ok bool
	if name, ok = mux.Vars(r)["name"]; !ok {
		http.Error(w, "A name must be provided (ex: /hero/kill/{name})", http.StatusBadRequest)
		return
	}

	mapOfHeroes := attemptToGetHeroData()
	if mapOfHeroes == nil {
		http.Error(w, resourceNotAvailableErr, http.StatusInternalServerError)
		return
	}

	var hero hero
	if hero, ok = mapOfHeroes[name]; !ok {
		heroMapChannel <- mapOfHeroes
		errMessage := fmt.Sprintf("Hero with name %q does not exist", name)
		http.Error(w, errMessage, http.StatusNotFound)
		return
	}
	if !hero.Alive {
		heroMapChannel <- mapOfHeroes
		errMessage := fmt.Sprintf("Hero with name %q has already been killed, and thus cannot be killed again", name)
		http.Error(w, errMessage, http.StatusBadRequest)
		return
	}

	hero.Alive = false
	mapOfHeroes[name] = hero
	heroMapChannel <- mapOfHeroes
	w.WriteHeader(http.StatusOK)
}

func heroRetire(w http.ResponseWriter, r *http.Request) {
	var name string
	var ok bool
	if name, ok = mux.Vars(r)["name"]; !ok {
		http.Error(w, "A name must be provided (ex: /hero/{name})", http.StatusBadRequest)
		return
	}

	mapOfHeroes := attemptToGetHeroData()
	if mapOfHeroes == nil {
		http.Error(w, resourceNotAvailableErr, http.StatusInternalServerError)
		return
	}

	var hero hero
	if hero, ok = mapOfHeroes[name]; !ok {
		heroMapChannel <- mapOfHeroes
		errMessage := fmt.Sprintf("Hero with name %q does not exist", name)
		http.Error(w, errMessage, http.StatusNotFound)
		return
	}
	if !hero.Alive {
		heroMapChannel <- mapOfHeroes
		errMessage := fmt.Sprintf("Hero with name %q has been killed, and thus cannot retire", name)
		http.Error(w, errMessage, http.StatusBadRequest)
		return
	}
	delete(mapOfHeroes, name)
	heroMapChannel <- mapOfHeroes
	w.WriteHeader(http.StatusOK)
}

func heroRest(w http.ResponseWriter, r *http.Request) {
	var name string
	var ok bool
	if name, ok = mux.Vars(r)["name"]; !ok {
		http.Error(w, "A name must be provided (ex: /hero/rest/{name})", http.StatusBadRequest)
		return
	}

	mapOfHeroes := attemptToGetHeroData()
	if mapOfHeroes == nil {
		http.Error(w, resourceNotAvailableErr, http.StatusInternalServerError)
		return
	}

	var hero hero
	if hero, ok = mapOfHeroes[name]; !ok {
		heroMapChannel <- mapOfHeroes
		errMessage := fmt.Sprintf("Hero with name %q does not exist", name)
		http.Error(w, errMessage, http.StatusNotFound)
		return
	}
	if !hero.Alive {
		heroMapChannel <- mapOfHeroes
		errMessage := fmt.Sprintf("Hero with name %q is dead, and thus can not rest", name)
		http.Error(w, errMessage, http.StatusBadRequest)
		return
	}
	if hero.Exhaustion == 0 {
		heroMapChannel <- mapOfHeroes
		errMessage := fmt.Sprintf("Hero with name %q does not need rest", name)
		http.Error(w, errMessage, http.StatusBadRequest)
		return
	}
	hero.Exhaustion--
	mapOfHeroes[name] = hero
	heroMapChannel <- mapOfHeroes
	w.WriteHeader(http.StatusOK)
}

func heroGet(w http.ResponseWriter, r *http.Request) {
	var name string
	var ok bool
	if name, ok = mux.Vars(r)["name"]; !ok {
		http.Error(w, "A name must be provided (ex: /hero/{name})", http.StatusBadRequest)
		return
	}

	mapOfHeroes := attemptToGetHeroData()
	if mapOfHeroes == nil {
		http.Error(w, resourceNotAvailableErr, http.StatusInternalServerError)
		return
	}

	var hero hero
	hero, ok = mapOfHeroes[name]
	// Return map to channel ASAP to prevent blocking other threads
	heroMapChannel <- mapOfHeroes
	if !ok {
		errMessage := fmt.Sprintf("Hero with name %q does not exist", name)
		http.Error(w, errMessage, http.StatusNotFound)
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

	mapOfHeroes := attemptToGetHeroData()
	if mapOfHeroes == nil {
		http.Error(w, resourceNotAvailableErr, http.StatusInternalServerError)
		return
	}

	if hero, heroExists := mapOfHeroes[hero.Name]; heroExists {
		heroMapChannel <- mapOfHeroes
		if hero.Alive {
			errMessage := fmt.Sprintf("Hero with name %q already exists", hero.Name)
			http.Error(w, errMessage, http.StatusConflict)
		} else {
			errMessage := fmt.Sprintf("A hero named %q once died valiantly in battle, and their name shall not be taken", hero.Name)
			http.Error(w, errMessage, http.StatusConflict)
		}
		return
	}
	hero.Alive = true
	mapOfHeroes[hero.Name] = hero
	heroMapChannel <- mapOfHeroes
	w.WriteHeader(http.StatusOK)
}

// This method will try to receive the map from the channel and then subsequently return it.
// If the channel fails to produce the map within three seconds, this method will stop trying
// to receive the map and simply return a nil value instead.
//
// If this method is successful in returning the map of hero data, then the code that follows
// must send the map back into channel at some point. Failure to do so will result in a loss
// of all data and the service will no longer be usable.
func attemptToGetHeroData() map[string]hero {
	select {
	case mapOfHeroes := <-heroMapChannel:
		return mapOfHeroes
	case <-time.After(3 * time.Second):
		return nil
	}
}

func linkRoutes(r *mux.Router) {
	// It might be more accurrate to make this a "PUT" due to it's idempotence
	r.HandleFunc("/hero", heroMake).Methods("POST")

	r.HandleFunc("/hero/{name}", heroGet).Methods("GET")

	// These are not "traditional PATCH" methods. No data expected in messages.
	r.HandleFunc("/hero/rest/{name}", heroRest).Methods("PATCH")
	r.HandleFunc("/hero/kill/{name}", heroKill).Methods("PATCH")

	r.HandleFunc("/hero/{name}", heroRetire).Methods("DELETE")

	r.HandleFunc("/calamity", handleCalamity).Methods("POST")
}

func main() {
	// Initalize channel & map to hold the heroes data
	heroMapChannel = make(chan map[string]hero, 1)
	heroMapChannel <- make(map[string]hero)

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

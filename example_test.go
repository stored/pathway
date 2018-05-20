package main

import (
	"fmt"
)

var UserMap = map[int]string{}
var UserID = 0

type USERS struct{}

func (USERS) Create(name string) int {
	UserMap[UserID] = name
	UserID += 1
	fmt.Println("created", name, "id", UserID-1)
	return UserID - 1
}

func (USERS) Delete(id int) {
	delete(UserMap, id)
}

func (USERS) Read(id int) string {
	return UserMap[id]
}

func (USERS) Update(id int, name string) {
	UserMap[id] = name
}

func Example() {
	AddBackend(USERS{}) // will panic if something goes wrong

	idRes, _ := Handler.CallBackendString("users/create", `"daniel"`)
	fmt.Println("CREATE:", idRes)

	idString := fmt.Sprintf(`%d`, idRes)

	res, _ := Handler.CallBackend("users/read/"+idString, nil)
	fmt.Println("READ:", res)

	Handler.CallBackendString("users/update/"+idString, `"galvez"`)

	res2, _ := Handler.CallBackend("users/read/"+idString, nil)
	fmt.Println("UPDATED:", res2)

	Handler.CallBackend("users/delete/"+idString, nil)

	res3, _ := Handler.CallBackend("users/read/"+idString, nil)
	fmt.Println("DELETED:", res3)

	// Output:
	// created daniel id 0
	// CREATE: 0
	// READ: daniel
	// UPDATED: galvez
	// DELETED:
}

package ubac

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ariary/AravisFS/pkg/filesystem"
)

// Provide all the resources within the fs in form of node list which represent the tree:
// each node  = (resourcename (encrypted), resource.Type)
// it also could have returned the content of each file, but it is like returning the whole fs (which is not a good idea)
// Node list is in json format
func Tree(filename string) string {

	resourcesList, err := GetResourceList(filename)
	if err != nil {
		log.Fatal(err)
	}
	resources := resourcesList.List

	nl := []filesystem.Node{}
	tree := filesystem.Tree{nl}

	// Tree construction
	for i := range resources {
		//Construct node and add it to the tree
		name := base64.StdEncoding.EncodeToString(resources[i].Name)
		node := filesystem.CreateNode(name, resources[i].Type)
		tree.AddNode(node)
	}

	// Tree to JSON
	treeJSON, _ := json.Marshal(resources)

	return string(treeJSON)
}

func PrintTree(filename string) {
	treeJSON := Tree(filename)

	fmt.Println(treeJSON)

}

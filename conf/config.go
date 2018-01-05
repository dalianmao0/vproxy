package conf

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func json2map(req string) (s map[string]interface{}, err error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(req), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func loadFile(fname string) (data string, err error) {
	fd, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	buffer := bufio.NewReader(fd)
	for {
		line, err := buffer.ReadString('\n')
		if err == io.EOF {
			if line[0] == '}' {
				data += line
			}

			break
		} else if line[0] == '#' {
			continue
		} else if err != nil {
			panic(err)
		}

		data += line
	}
	// fmt.Println("File content: ", data)

	return data, nil
}

// Load the config from file fname
func Load(fname string) (m map[string]string, e error) {
	fdata, err := loadFile(fname)
	if err != nil {
		fmt.Println("Read config.json failed.", err)
		return nil, err
	}

	configs, err := json2map(fdata)
	if err == nil {
		// convert map[string]interface{} to map[string]string
		mm := make(map[string]string)
		for k := range configs {
			mm[k] = configs[k].(string)
		}

		return mm, nil
	}

	return nil, err
}

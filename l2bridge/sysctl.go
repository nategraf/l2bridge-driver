package l2bridge

import (
	"io/ioutil"
)

//Gets the value of the kernel parameters located at the given path
func getSysBoolParam(path string) (bool, error) {
	enabled := false
	line, err := ioutil.ReadFile(path)
	if err != nil {
		return false, err
	}
	if len(line) > 0 {
		enabled = line[0] == '1'
	}
	return enabled, err
}

//Sets the value of the kernel parameter located at the given path
func setSysBoolParam(path string, on bool) error {
	value := byte('0')
	if on {
		value = byte('1')
	}
	return ioutil.WriteFile(path, []byte{value, '\n'}, 0644)
}

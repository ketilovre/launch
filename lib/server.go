package launch

import (
	"fmt"
	"os"
)

func CheckServerFile() error {
	file, err := os.Open("./server")

	if err != nil {
		return fmt.Errorf("Can't find or open your 'server' file: %v\n", err)
	}

	info, err := file.Stat()

	if err != nil {
		return fmt.Errorf("Can't stat your 'server' file: %v\n", err)
	}

	if (info.Mode() & 0001) == 0 {
		if err := file.Chmod(info.Mode() | 0111); err != nil {
			return fmt.Errorf(
				"Your 'server' file is not executable. An error occured while trying to update permissions: %v\n",
				err,
			)
		}
		fmt.Println("Making the 'server' file executable")
	}

	return nil
}

func CreateServerFile() error {
	file, err := os.Create("server")
	if err != nil {
		return fmt.Errorf("Couldn't create server file: %v", err)
	}

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Couldn't stat newly created server file: %v\n", err)
	}

	// Make it executable.
	err = file.Chmod(stat.Mode() | 0111)
	if err != nil {
		return fmt.Errorf("Couldn't make newly created server file executable: %v\n", err)
	}

	return nil
}

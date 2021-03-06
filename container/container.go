package container

import (
	"fmt"
	"strings"

	w "github.com/ibmjstart/cf-object-storage/writer"
	"github.com/ibmjstart/swiftlygo/auth"
	"github.com/ncw/swift"
)

// shortHeaders define shortcuts for header input.
var shortHeaders = map[string]string{
	"-gr":    "X-Container-Read:.r:*",
	"-rm-gr": "X-Remove-Container-Read:1",
}

// ShowContainers displays the containers in a given Object Storage service.
func ShowContainers(dest auth.Destination, writer *w.ConsoleWriter, args []string) (string, error) {
	writer.SetCurrentStage("Displaying containers")

	serviceName := args[2]

	containers, err := dest.(*auth.SwiftDestination).SwiftConnection.ContainerNamesAll(nil)
	if err != nil {
		return "", fmt.Errorf("Failed to get containers: %s", err)
	}

	return fmt.Sprintf("\r%s%s\n\nContainers in OS %s: %v\n", w.ClearLine, w.Green("OK"), serviceName, containers), nil
}

// GetContainerInfo displays metadata for a given container.
func GetContainerInfo(dest auth.Destination, writer *w.ConsoleWriter, args []string) (string, error) {
	writer.SetCurrentStage("Fetching container info")

	container := args[3]
	containerInfo, headers, err := dest.(*auth.SwiftDestination).SwiftConnection.Container(container)
	if err != nil {
		return "", fmt.Errorf("Failed to get container info for container %s: %s", container, err)
	}

	retval := fmt.Sprintf("\r%s%s\n\nName: %s\nnumber of objects: %d\nSize: %d bytes\nHeaders:", w.ClearLine, w.Green("OK"), containerInfo.Name, containerInfo.Count, containerInfo.Bytes)
	for k, h := range headers {
		retval += fmt.Sprintf("\n\tName: %s Value: %s", k, h)
	}
	retval += fmt.Sprintf("\n")

	return retval, nil
}

// MakeContainer creates a new container.
func MakeContainer(dest auth.Destination, writer *w.ConsoleWriter, args []string) (string, error) {
	writer.SetCurrentStage("Creating container")

	headerMap := make(map[string]string)
	serviceName := args[2]
	container := args[3]
	headers := args[4:]

	for _, h := range headers {
		hFromMap, found := shortHeaders[h]
		if found {
			h = hFromMap
		}

		headerPair := strings.SplitN(h, ":", 2)
		if len(headerPair) != 2 {
			return "", fmt.Errorf("Unable to parse headers (must use format header-name:header-value)")
		}

		headerMap[headerPair[0]] = headerPair[1]
	}

	swiftHeader := swift.Headers(headerMap)

	err := dest.(*auth.SwiftDestination).SwiftConnection.ContainerCreate(container, swiftHeader)
	if err != nil {
		return "", fmt.Errorf("Failed to create container: %s", err)
	}

	return fmt.Sprintf("\r%s%s\n\nCreated container %s in OS %s\n", w.ClearLine, w.Green("OK"), container, serviceName), nil
}

// DeleteContainer removes a container and all of its contents.
func DeleteContainer(dest auth.Destination, writer *w.ConsoleWriter, args []string) (string, error) {
	serviceName := args[2]
	container := args[3]

	if len(args) == 5 && args[4] == "-f" {
		writer.SetCurrentStage("Deleting objects in container")

		objects, err := dest.(*auth.SwiftDestination).SwiftConnection.ObjectNamesAll(container, nil)
		if err != nil {
			return "", fmt.Errorf("Failed to get objects to delete: %s", err)
		}

		for _, rmObject := range objects {
			err = dest.(*auth.SwiftDestination).SwiftConnection.ObjectDelete(container, rmObject)
			if err != nil {
				return "", fmt.Errorf("Failed to delete object %s: %s", rmObject, err)
			}
		}
	}

	writer.SetCurrentStage("Deleting container")

	err := dest.(*auth.SwiftDestination).SwiftConnection.ContainerDelete(container)
	if err != nil {
		return "", fmt.Errorf("Failed to delete container: %s", err)
	}

	return fmt.Sprintf("\r%s%s\n\nDeleted container %s from OS %s\n", w.ClearLine, w.Green("OK"), container, serviceName), nil
}

// UpdateContainer updates a containers metadata.
func UpdateContainer(dest auth.Destination, writer *w.ConsoleWriter, args []string) (string, error) {
	writer.SetCurrentStage("Updating container")

	serviceName := args[2]
	container := args[3]

	_, err := GetContainerInfo(dest, writer, args)
	if err != nil {
		return "", fmt.Errorf("Failed to get container %s: %s", container, err)
	}

	_, err = MakeContainer(dest, writer, args)
	if err != nil {
		return "", fmt.Errorf("Failed to make container: %s", err)
	}

	return fmt.Sprintf("\r%s%s\n\nUpdated container %s in OS %s\n", w.ClearLine, w.Green("OK"), container, serviceName), nil
}

// RenameContainer renames a container.
func RenameContainer(dest auth.Destination, writer *w.ConsoleWriter, args []string) (string, error) {
	writer.SetCurrentStage("Renaming container")

	container := args[3]
	newContainer := args[4]

	_, headers, err := dest.(*auth.SwiftDestination).SwiftConnection.Container(container)
	if err != nil {
		return "", fmt.Errorf("Failed to get container %s: %s", container, err)
	}

	headersArg := make([]string, 0)
	for header, val := range headers {
		headersArg = append(headersArg, header+":"+val)
	}

	makeArg := append(args[:3], append([]string{newContainer}, headersArg...)...)
	_, err = MakeContainer(dest, writer, makeArg)
	if err != nil {
		return "", fmt.Errorf("Failed to make container: %s", err)
	}

	writer.SetCurrentStage("Renaming container")

	objects, err := dest.(*auth.SwiftDestination).SwiftConnection.ObjectNamesAll(container, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to get objects to move: %s", err)
	}

	for _, mvObject := range objects {
		err := dest.(*auth.SwiftDestination).SwiftConnection.ObjectMove(container, mvObject, newContainer, mvObject)
		if err != nil {
			return "", fmt.Errorf("Failed to move object %s: %s", mvObject, err)
		}
	}

	_, err = DeleteContainer(dest, writer, args[:4])
	if err != nil {
		return "", fmt.Errorf("Failed to delete container: %s", err)
	}

	return fmt.Sprintf("\r%s%s\n\nRenamed container %s to %s\n", w.ClearLine, w.Green("OK"), container, newContainer), nil
}

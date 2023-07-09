package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/guptarohit/asciigraph"
)

type memoryStats struct {
	Usage int `json:"usage"`
	Limit int `json:"limit"`
}

type dockerStats struct {
	MemStats memoryStats `json:"memory_stats"`
}

// Returns a stream of container stats json for the container with
// the provided name.
func getContainerStats(containerName string) types.ContainerStats {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)

	containers, err := cli.ContainerList(
		context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	cont_id := new(bytes.Buffer)

	// Get ID of container with matching name
	for _, container := range containers {
		var stats map[string]interface{}
		resp, err := cli.ContainerStatsOneShot(ctx, container.ID)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&stats)

		if stats["name"] == containerName {
			fmt.Fprintf(cont_id, "%s", stats["id"])
		}
	}

	if cont_id == nil {
		panic("Help")
	}

	resp, err := cli.ContainerStats(ctx, cont_id.String(), true)
	if err != nil {
		panic(err)
	}

	return resp
}

// Appends value to the buffer data and truncates from the
// front whenever the internal length threshold has been reached
func insertBuffer(value float64, data []float64) []float64 {
	plotLength := 20
	if len(data) > plotLength {
		data = (data)[1:]
	}
	return append(data, value)
}

func getGraphOpts(contName string) []asciigraph.Option {
    return []asciigraph.Option{
        asciigraph.Height(15),
        asciigraph.Width(75),
		asciigraph.LowerBound(0),
        asciigraph.SeriesColors(
			asciigraph.LightCoral,
			asciigraph.Turquoise,
		),
        asciigraph.Caption(
			fmt.Sprintf(
				"Memory usage for container: %s",
				contName,
			),
		),
    }
}

func byteToGiB(val float64) float64 {
    return val / math.Pow(1024, 3)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s container-name\n", os.Args[0])
		os.Exit(1)
	}

	containerName := fmt.Sprintf("/%s", os.Args[1])
	containerStats := getContainerStats(containerName)

	var stats dockerStats
	dec := json.NewDecoder(containerStats.Body)

	usageSeries := make([]float64, 0)
	limitSeries := make([]float64, 0)
	
	for dec.More() {
		err := dec.Decode(&stats)
	
		if err != nil {
			panic(err)
		}
		usageGiB := byteToGiB(float64(stats.MemStats.Usage))
        limitGiB := byteToGiB(float64(stats.MemStats.Limit))
		usageSeries = insertBuffer(usageGiB, usageSeries)
		limitSeries = insertBuffer(limitGiB, limitSeries)
		data := [][]float64{limitSeries, usageSeries}

        asciigraph.Clear()
		graph := asciigraph.PlotMany(
            data,
            getGraphOpts(containerName)...,
        ) 
		fmt.Println(graph)
    }
}

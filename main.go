package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/sascha-andres/reuse/flag"
)

var (
	public, all, jsonOutput bool
	logLevel                uint
)

type (

	// ip represents a network interface and its associated IP address.
	ip struct {

		// Address represents the IP address associated with a network interface.
		Address string

		// Interface represents the name of the network interface associated with the IP address.
		Interface string
	}

	// ips represents a collection of ip instances, each containing details about a network interface and its IP address.
	ips []*ip
)

// String returns a formatted string representation of the ip, combining its Address and Interface fields.
func (i ip) String() string {
	return fmt.Sprintf("%s\t%s", i.Address, i.Interface)
}

// main is the entry point of the application, parsing flags to determine the mode of operation and executing the run function.
func main() {
	flag.SetEnvPrefix("IPS")
	flag.BoolVar(&public, "p", false, "print public ip only, exclusive to -a")
	flag.BoolVar(&all, "a", false, "print all ip, exclusive to -ap")
	flag.BoolVar(&jsonOutput, "json", false, "output as JSON")
	flag.UintVar(&logLevel, "l", 0, "log level")
	flag.Parse()

	var handlerOpts *slog.HandlerOptions
	switch logLevel {
	case 0:
		handlerOpts = &slog.HandlerOptions{Level: slog.LevelWarn}
	case 1:
		handlerOpts = &slog.HandlerOptions{Level: slog.LevelInfo}
	case 2:
		handlerOpts = &slog.HandlerOptions{Level: slog.LevelDebug}
	default:
		handlerOpts = &slog.HandlerOptions{Level: slog.LevelInfo}
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, handlerOpts)).With("project", "ips")
	slog.SetDefault(logger)

	logger.Debug(
		"starting",
		slog.Any("public", public),
		slog.Any("all", all),
		slog.Any("json", jsonOutput),
		slog.Any("logLevel", logLevel),
	)

	if err := run(logger); err != nil {
		os.Exit(1)
	}
}

// run retrieves IP addresses, logs errors if retrieval fails, and outputs the addresses in plain text or JSON format.
func run(logger *slog.Logger) error {
	// get ips
	ips, err := getIpAddresses()
	if err != nil {
		logger.Error("could not get ip addresses", "err", err)
		return err
	}
	if jsonOutput {
		data, err := json.Marshal(ips)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else {
		for _, i := range ips {
			fmt.Println(i)
		}
	}
	return nil
}

// getIpAddresses retrieves a list of IP addresses for all available network interfaces.
// If the public flag is set, it includes the public IP address.
// Returns a collection of IP instances and an error if any occurs during retrieval.
func getIpAddresses() (ips, error) {
	ips := make(ips, 0)
	if public || all {
		publicIp, err := getPublicIp()
		if err != nil {
			return ips, err
		}
		if publicIp != nil {
			ips = append(ips, publicIp)
		}
	}
	if !all && public {
		return ips, nil
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return ips, err
	}
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			return ips, err
		}
		if addrs == nil || len(addrs) == 0 {
			continue
		}
		for _, addr := range addrs {
			ips = append(ips, &ip{
				Address:   addr.String(),
				Interface: i.Name,
			})
		}
	}
	return ips, nil
}

// getPublicIp retrieves the public IP address of the system using an external service and returns it as an ip instance.
// Returns an error if the request fails or the response can't be processed.
func getPublicIp() (*ip, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://wtfismyip.com/text", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/8.7.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &ip{
		Address:   strings.TrimSpace(string(body)),
		Interface: "public",
	}, nil
}

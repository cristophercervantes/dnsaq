package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// DNSConfig holds configuration for the DNS enumerator
type DNSConfig struct {
	Resolvers     []string
	RateLimit     int
	Timeout       time.Duration
	WildcardCheck bool
	Verbose       bool
	OutputFile    string
}

// DNSEnumerator handles DNS resolution and enumeration
type DNSEnumerator struct {
	Config      *DNSConfig
	client      *dns.Client
	wildcardIPs map[string]bool
	mutex       sync.Mutex
	outputFile  *os.File
}

// NewDNSEnumerator creates a new DNS enumerator instance
func NewDNSEnumerator(config *DNSConfig) (*DNSEnumerator, error) {
	client := &dns.Client{
		Timeout: config.Timeout,
		Net:     "udp",
	}

	enumerator := &DNSEnumerator{
		Config:      config,
		client:      client,
		wildcardIPs: make(map[string]bool),
	}

	// Open output file if specified
	if config.OutputFile != "" {
		file, err := os.OpenFile(config.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("error opening output file: %v", err)
		}
		enumerator.outputFile = file
	}

	return enumerator, nil
}

// Close cleans up resources
func (d *DNSEnumerator) Close() {
	if d.outputFile != nil {
		d.outputFile.Close()
	}
}

// LoadResolversFromFile loads DNS resolvers from a file
func LoadResolversFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var resolvers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		resolver := strings.TrimSpace(scanner.Text())
		if resolver != "" && !strings.HasPrefix(resolver, "#") {
			// Ensure resolver has port if not already included
			if !strings.Contains(resolver, ":") {
				resolver = net.JoinHostPort(resolver, "53")
			}
			resolvers = append(resolvers, resolver)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return resolvers, nil
}

// Resolve performs a DNS lookup for a domain
func (d *DNSEnumerator) Resolve(domain string) ([]string, error) {
	msg := &dns.Msg{}
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)

	// Try each resolver until we get a response
	for _, resolver := range d.Config.Resolvers {
		resp, _, err := d.client.Exchange(msg, resolver)
		if err != nil {
			if d.Config.Verbose {
				fmt.Fprintf(os.Stderr, "Resolver %s failed: %v\n", resolver, err)
			}
			continue // Try next resolver
		}

		if resp.Rcode != dns.RcodeSuccess {
			return nil, fmt.Errorf("DNS error: %v", resp.Rcode)
		}

		var ips []string
		for _, answer := range resp.Answer {
			if a, ok := answer.(*dns.A); ok {
				ips = append(ips, a.A.String())
			}
		}
		return ips, nil
	}

	return nil, fmt.Errorf("all resolvers failed")
}

// DetectWildcard checks if a domain has wildcard DNS configured
func (d *DNSEnumerator) DetectWildcard(domain string) {
	if !d.Config.WildcardCheck {
		return
	}

	// Test with random subdomains that likely don't exist
	testSubdomains := []string{
		fmt.Sprintf("rand%d-%d", time.Now().Unix(), os.Getpid()),
		"probably-does-not-exist-123",
		"test-subdomain-wildcard-456",
	}

	for _, sub := range testSubdomains {
		testDomain := sub + "." + domain
		ips, err := d.Resolve(testDomain)
		if err == nil && len(ips) > 0 {
			d.mutex.Lock()
			for _, ip := range ips {
				d.wildcardIPs[ip] = true
			}
			d.mutex.Unlock()
		}
	}

	if len(d.wildcardIPs) > 0 && d.Config.Verbose {
		fmt.Fprintf(os.Stderr, "[!] Wildcard DNS detected. These IPs will be filtered: %v\n", d.getWildcardIPs())
	}
}

func (d *DNSEnumerator) getWildcardIPs() []string {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	ips := make([]string, 0, len(d.wildcardIPs))
	for ip := range d.wildcardIPs {
		ips = append(ips, ip)
	}
	return ips
}

func (d *DNSEnumerator) isWildcardResponse(ips []string) bool {
	if len(d.wildcardIPs) == 0 {
		return false
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	for _, ip := range ips {
		if d.wildcardIPs[ip] {
			return true
		}
	}
	return false
}

// WriteOutput writes results to both stdout and output file (if specified)
func (d *DNSEnumerator) WriteOutput(result string) {
	fmt.Println(result)
	if d.outputFile != nil {
		d.outputFile.WriteString(result + "\n")
	}
}

// ProcessDomain resolves a domain and sends results to the channel
func (d *DNSEnumerator) ProcessDomain(domain string, results chan<- string) {
	ips, err := d.Resolve(domain)
	if err != nil {
		if d.Config.Verbose {
			fmt.Fprintf(os.Stderr, "Error resolving %s: %v\n", domain, err)
		}
		return
	}

	// Skip wildcard responses if enabled
	if d.Config.WildcardCheck && d.isWildcardResponse(ips) {
		if d.Config.Verbose {
			fmt.Fprintf(os.Stderr, "Filtered wildcard response for %s: %v\n", domain, ips)
		}
		return
	}

	results <- fmt.Sprintf("%s [%s]", domain, strings.Join(ips, ", "))
}

// EnumerateFromReader processes domains from a reader (stdin or file)
func (d *DNSEnumerator) EnumerateFromReader(reader *bufio.Reader) {
	// Rate limiting
	limiter := time.Tick(time.Second / time.Duration(d.Config.RateLimit))
	results := make(chan string, 100)
	
	// Process results
	go func() {
		for result := range results {
			d.WriteOutput(result)
		}
	}()

	var wg sync.WaitGroup
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain == "" {
			continue
		}
		
		// Extract base domain for wildcard detection
		if d.Config.WildcardCheck {
			parts := strings.Split(domain, ".")
			if len(parts) >= 2 {
				baseDomain := parts[len(parts)-2] + "." + parts[len(parts)-1]
				d.DetectWildcard(baseDomain)
			}
		}
		
		<-limiter
		wg.Add(1)
		go func(dmn string) {
			defer wg.Done()
			d.ProcessDomain(dmn, results)
		}(domain)
	}

	wg.Wait()
	close(results)
}

// Bruteforce performs subdomain brute-forcing
func (d *DNSEnumerator) Bruteforce(domain string, wordlistPath string) {
	d.DetectWildcard(domain)

	file, err := os.Open(wordlistPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening wordlist: %v\n", err)
		return
	}
	defer file.Close()

	limiter := time.Tick(time.Second / time.Duration(d.Config.RateLimit))
	results := make(chan string, 100)
	
	// Process results
	go func() {
		for result := range results {
			d.WriteOutput(result)
		}
	}()

	var wg sync.WaitGroup
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		sub := strings.TrimSpace(scanner.Text())
		if sub == "" {
			continue
		}

		fullDomain := sub + "." + domain
		<-limiter
		wg.Add(1)
		go func(dmn string) {
			defer wg.Done()
			d.ProcessDomain(dmn, results)
		}(fullDomain)
	}

	wg.Wait()
	close(results)

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading wordlist: %v\n", err)
	}
}

func main() {
	var (
		domain        = flag.String("d", "", "Domain to brute-force")
		wordlist      = flag.String("w", "", "Wordlist for brute-force")
		resolverFile  = flag.String("r", "", "File containing DNS resolvers (one per line)")
		resolverList  = flag.String("resolvers", "8.8.8.8:53,1.1.1.1:53", "Comma-separated list of DNS resolvers")
		rateLimit     = flag.Int("rate", 10, "Queries per second")
		timeout       = flag.Int("t", 2, "Timeout in seconds")
		noWildcard    = flag.Bool("no-wildcard", false, "Disable wildcard detection")
		verbose       = flag.Bool("v", false, "Verbose output")
		version       = flag.Bool("version", false, "Show version information")
		outputFile    = flag.String("o", "", "Output file to save results")
	)
	flag.Parse()

	if *version {
		fmt.Println("DNS Tool v1.0.0")
		os.Exit(0)
	}

	// Load resolvers
	var resolvers []string
	if *resolverFile != "" {
		fileResolvers, err := LoadResolversFromFile(*resolverFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading resolvers from file: %v\n", err)
			os.Exit(1)
		}
		resolvers = fileResolvers
	} else {
		resolvers = strings.Split(*resolverList, ",")
	}

	// Validate we have resolvers
	if len(resolvers) == 0 {
		fmt.Fprintln(os.Stderr, "No DNS resolvers specified")
		os.Exit(1)
	}

	config := &DNSConfig{
		Resolvers:     resolvers,
		RateLimit:     *rateLimit,
		Timeout:       time.Duration(*timeout) * time.Second,
		WildcardCheck: !*noWildcard,
		Verbose:       *verbose,
		OutputFile:    *outputFile,
	}

	enumerator, err := NewDNSEnumerator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing DNS enumerator: %v\n", err)
		os.Exit(1)
	}
	defer enumerator.Close()

	if *domain != "" && *wordlist != "" {
		// Brute-force subdomains
		enumerator.Bruteforce(*domain, *wordlist)
	} else {
		// Read from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Data is being piped in
			enumerator.EnumerateFromReader(bufio.NewReader(os.Stdin))
		} else {
			fmt.Fprintln(os.Stderr, "DNS Tool - Fast DNS resolution and subdomain enumeration")
			fmt.Fprintln(os.Stderr, "Usage: dns-tool -d example.com -w wordlist.txt -r resolvers.txt")
			fmt.Fprintln(os.Stderr, "       subfinder -d example.com | dns-tool -r resolvers.txt")
			fmt.Fprintln(os.Stderr, "       cat domains.txt | dns-tool -r resolvers.txt")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Options:")
			flag.PrintDefaults()
			os.Exit(1)
		}
	}
}

package faker

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// schemaType extracts a single type string, tolerating kin-openapi's *Types
// (which is a slice) and nullable unions.
func schemaType(s *openapi3.Schema) string {
	if s.Type == nil {
		return ""
	}
	for _, t := range s.Type.Slice() {
		if t != "null" {
			return t
		}
	}
	return ""
}

// byFormat maps OpenAPI string formats to realistic generators.
func (f *Faker) byFormat(format string) (any, bool) {
	switch format {
	case "uuid":
		return f.uuid(), true
	case "email", "idn-email":
		return f.email(), true
	case "date-time":
		return f.recentDate(), true
	case "date":
		return f.recentDate()[:10], true
	case "uri", "url", "iri":
		return f.url(), true
	case "hostname":
		return "api." + f.pick(domains), true
	case "ipv4":
		return f.ipv4(), true
	case "byte":
		return "U29tZSBtb2NrIGRhdGE=", true
	case "password":
		return "P@ssw0rd!" + f.uuid()[:6], true
	}
	return "", false
}

// byGenerator resolves an x-mock vendor-extension generator name.
func (f *Faker) byGenerator(name string) (any, bool) {
	switch strings.ToLower(name) {
	case "uuid":
		return f.uuid(), true
	case "email":
		return f.email(), true
	case "name", "fullname":
		return f.fullName(), true
	case "firstname":
		return f.pick(firstNames), true
	case "lastname":
		return f.pick(lastNames), true
	case "phone":
		return f.phone(), true
	case "city":
		return f.pick(cities), true
	case "country":
		return f.pick(countries), true
	case "company":
		return f.pick(companies), true
	case "url":
		return f.url(), true
	case "money", "price":
		return f.money(), true
	case "date", "datetime":
		return f.recentDate(), true
	case "sentence":
		return f.sentence(), true
	case "word":
		return f.pick(words), true
	case "bool", "boolean":
		return f.r.Intn(2) == 0, true
	}
	return nil, false
}

// heuristic infers a realistic string from the property name.
func (f *Faker) heuristic(name, format string) any {
	n := normalize(name)
	switch {
	case n == "":
		return f.sentence()
	case has(n, "email"):
		return f.email()
	case n == "id" || strings.HasSuffix(n, "id") || strings.HasSuffix(n, "uuid"):
		return f.uuid()
	case has(n, "firstname"), n == "fname":
		return f.pick(firstNames)
	case has(n, "lastname"), has(n, "surname"), n == "lname":
		return f.pick(lastNames)
	case has(n, "username"), n == "user", n == "handle", n == "login":
		return strings.ToLower(f.pick(firstNames)) + f.numStr(2)
	case has(n, "name") && !has(n, "filename"):
		return f.fullName()
	case has(n, "phone"), has(n, "mobile"), has(n, "tel"):
		return f.phone()
	case has(n, "company"), has(n, "organization"), has(n, "org"), has(n, "tenant"):
		return f.pick(companies)
	case has(n, "country"):
		return f.pick(countries)
	case has(n, "city"), has(n, "town"):
		return f.pick(cities)
	case has(n, "state"), has(n, "province"), has(n, "region"):
		return f.pick(states)
	case has(n, "street"), has(n, "address"):
		return f.address()
	case has(n, "zip"), has(n, "postal"):
		return f.numStr(5)
	case has(n, "url"), has(n, "link"), has(n, "website"), has(n, "href"):
		return f.url()
	case has(n, "avatar"), has(n, "image"), has(n, "photo"), has(n, "picture"), has(n, "thumbnail"):
		return "https://i.pravatar.cc/150?u=" + f.uuid()[:8]
	case has(n, "slug"):
		return strings.ToLower(f.pick(words)) + "-" + strings.ToLower(f.pick(words))
	case has(n, "color"), has(n, "colour"):
		return f.pick(colors)
	case has(n, "currency"):
		return f.pick(currencies)
	case has(n, "status"), has(n, "state"):
		return f.pick(statuses)
	case has(n, "gender"):
		return f.pick([]string{"male", "female", "other"})
	case has(n, "title"), has(n, "subject"), has(n, "headline"):
		return f.titleCase(f.sentence())
	case has(n, "description"), has(n, "summary"), has(n, "bio"), has(n, "comment"), has(n, "message"), has(n, "content"), has(n, "body"), has(n, "note"):
		return f.sentence()
	case has(n, "token"), has(n, "secret"), has(n, "key"), has(n, "hash"):
		return f.token()
	case has(n, "date"), has(n, "time"), strings.HasSuffix(n, "at"):
		return f.recentDate()
	case has(n, "lat"):
		return "" // handled numerically; string fallback below
	default:
		return f.pick(words)
	}
}

// heuristicInt infers realistic integers from the property name.
func (f *Faker) heuristicInt(name string) (int, bool) {
	n := normalize(name)
	switch {
	case n == "id" || strings.HasSuffix(n, "id"):
		return 1 + f.r.Intn(99999), true
	case has(n, "age"):
		return 18 + f.r.Intn(60), true
	case has(n, "year"):
		return 1990 + f.r.Intn(40), true
	case has(n, "count"), has(n, "quantity"), has(n, "qty"), has(n, "total"), has(n, "number"), has(n, "num"):
		return f.r.Intn(500), true
	case has(n, "page"):
		return 1 + f.r.Intn(20), true
	case has(n, "rating"), has(n, "score"):
		return 1 + f.r.Intn(5), true
	case has(n, "percent"), has(n, "progress"):
		return f.r.Intn(101), true
	}
	return 0, false
}

func isMoney(name string) bool {
	n := normalize(name)
	for _, k := range []string{"price", "amount", "cost", "balance", "total", "fee", "salary", "revenue", "subtotal"} {
		if has(n, k) {
			return true
		}
	}
	return false
}

// --- string helpers ---

func normalize(s string) string {
	s = strings.ToLower(s)
	return strings.NewReplacer("_", "", "-", "", " ", "").Replace(s)
}

func has(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func (f *Faker) pick(list []string) string { return list[f.r.Intn(len(list))] }

func (f *Faker) numStr(n int) string {
	const digits = "0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = digits[f.r.Intn(10)]
	}
	return string(b)
}

func (f *Faker) uuid() string {
	const hex = "0123456789abcdef"
	b := make([]byte, 36)
	for i := range b {
		switch i {
		case 8, 13, 18, 23:
			b[i] = '-'
		case 14:
			b[i] = '4'
		default:
			b[i] = hex[f.r.Intn(16)]
		}
	}
	return string(b)
}

func (f *Faker) token() string {
	const cs = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = cs[f.r.Intn(len(cs))]
	}
	return string(b)
}

func (f *Faker) email() string {
	return strings.ToLower(f.pick(firstNames)) + "." + strings.ToLower(f.pick(lastNames)) + f.numStr(2) + "@" + f.pick(domains)
}

func (f *Faker) fullName() string { return f.pick(firstNames) + " " + f.pick(lastNames) }

func (f *Faker) phone() string {
	return "+1 (" + f.numStr(3) + ") " + f.numStr(3) + "-" + f.numStr(4)
}

func (f *Faker) url() string {
	return "https://" + f.pick(domains) + "/" + strings.ToLower(f.pick(words))
}

func (f *Faker) ipv4() string {
	return f.numStrRange(1, 255) + "." + f.numStrRange(0, 255) + "." + f.numStrRange(0, 255) + "." + f.numStrRange(1, 255)
}

func (f *Faker) numStrRange(lo, hi int) string {
	return itoa(lo + f.r.Intn(hi-lo+1))
}

func (f *Faker) address() string {
	return itoa(1+f.r.Intn(9999)) + " " + f.pick(lastNames) + " " + f.pick([]string{"St", "Ave", "Rd", "Blvd", "Ln"})
}

func (f *Faker) sentence() string {
	n := 6 + f.r.Intn(8)
	parts := make([]string, n)
	for i := range parts {
		parts[i] = f.pick(words)
	}
	s := strings.Join(parts, " ")
	return strings.ToUpper(s[:1]) + s[1:] + "."
}

func (f *Faker) titleCase(s string) string {
	s = strings.TrimSuffix(s, ".")
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

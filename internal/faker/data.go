package faker

// Curated word banks. Small on purpose — deterministic seeding makes a modest
// list feel varied across fields, and it keeps the binary dependency-free.

var firstNames = []string{
	"Ada", "Chidi", "Amara", "Liam", "Sofia", "Kwame", "Mei", "Noah", "Zara", "Tunde",
	"Olivia", "Ravi", "Ingrid", "Mateo", "Aisha", "Hiro", "Lucas", "Fatima", "Elena", "Bayo",
}

var lastNames = []string{
	"Okafor", "Smith", "Nguyen", "Garcia", "Adeyemi", "Kim", "Johansson", "Rossi", "Khan", "Mensah",
	"Silva", "Patel", "Andersen", "Costa", "Ibrahim", "Tanaka", "Brown", "Diallo", "Petrov", "Eze",
}

var domains = []string{
	"example.com", "acme.io", "test.dev", "mail.co", "corp.net", "heirs.africa", "converge.app",
}

var companies = []string{
	"Acme Corp", "Globex", "Initech", "Umbrella Ltd", "Heirs Holdings", "Converge Systems",
	"Stark Industries", "Wayne Enterprises", "Soylent", "Hooli",
}

var cities = []string{
	"Lagos", "Nairobi", "Accra", "Cairo", "London", "Berlin", "Tokyo", "São Paulo",
	"Toronto", "Mumbai", "Dubai", "Singapore",
}

var states = []string{
	"Lagos", "Abuja", "California", "Texas", "Ontario", "Bavaria", "Maharashtra", "Gauteng",
}

var countries = []string{
	"Nigeria", "Kenya", "Ghana", "United States", "United Kingdom", "Germany",
	"Japan", "Brazil", "Canada", "India", "South Africa", "Singapore",
}

var statuses = []string{
	"active", "pending", "inactive", "archived", "draft", "completed", "failed", "processing",
}

var colors = []string{
	"red", "green", "blue", "amber", "violet", "teal", "slate", "rose", "indigo", "emerald",
}

var currencies = []string{"USD", "NGN", "EUR", "GBP", "KES", "GHS", "ZAR", "JPY"}

var words = []string{
	"lorem", "ipsum", "dolor", "amet", "consectetur", "adipiscing", "elit", "tempor",
	"labore", "magna", "aliqua", "veniam", "nostrud", "ullamco", "laboris", "aliquip",
	"commodo", "consequat", "aute", "irure", "voluptate", "cillum", "fugiat", "pariatur",
}

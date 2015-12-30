package format

import (
	"log"
	"strings"
	"time"
)

const (
	StandardDateLayout = "2006-01-02" // YYYY-DD-MM
	EanDateLayout      = "01/02/2006" // MM/DD/YYYY
	CacheDateLayout    = "01022006"   // MMDDYY
)

func TimeInStringsOut(layout string, t1, t2 time.Time) (string, string) {
	return t1.Format(layout), t2.Format(layout)
}

func StringInTimeOut(layout, s1, s2 string) (time.Time, time.Time) {
	t1, err1 := time.Parse(layout, s1)
	if err1 != nil {
		log.Println("Error parsing string1", err1)
	}
	t2, err2 := time.Parse(layout, s2)
	if err2 != nil {
		log.Println("Error parsing string2", err1)
	}

	return t1, t2
}

func StringsFromTimeToKey(s1, s2 string) string {
	ss1 := strings.Join(strings.Split(s1, "/"), "")
	ss2 := strings.Join(strings.Split(s2, "/"), "")
	return strings.Join([]string{ss1, ss2}, "-")
}

func StringSplitToTimes(layout, s string) (time.Time, time.Time) {
	szero := strings.Split(s, "-")
	return StringInTimeOut(layout, szero[0], szero[1])
}

/*  Move this to a test!!
func main() {

	s := "01/10/2014"
	s2 := "01/17/2014"
	c := strings.Join(strings.Split(s, "/"), "")
	e := strings.Join(strings.Split(s2, "/"), "")
	k := strings.Join([]string{c, e}, "-")
	fmt.Println(k)

	time1 := time.Now()
	time2 := time1.AddDate(0, 0, 7)
	a, d := timeInStringOut(dateLayout, time1, time2)
	fmt.Printf("%v, %v\n", a, d)

	z, y := stringInTimeOut(dateLayout, a, d)
	fmt.Printf("%v, %v\n", z, y)
}
*/

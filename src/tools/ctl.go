package main

import (
	"encoding/base64"
	"path/filepath"
	"io/ioutil"
	"net/http"
	"strings"
	"strconv"
	"regexp"
	"time"
	"flag"
	"fmt"
	"os"

	"../common"
	"../common/http"
	"../apis/apps"
)

type LoginInfo struct {
	Host		string		`yaml:"host"`
	Port		string		`yaml:"port"`
	Token		string		`yaml:"token"`
	User		string		`yaml:"user"`
	Pass		string		`yaml:"pass"`
}

type YAMLConf struct {
	Login		LoginInfo	`yaml:"login"`
	TLS		bool		`yaml:"tls"`
	Certs		string		`yaml:"x509crtfile"`
}

var conf YAMLConf

func fatal(err error) {
	fmt.Printf("ERROR: %s\n", err.Error())
	os.Exit(1)
}

func gateProto() string {
	if conf.TLS {
		return "https"
	} else {
		return "http"
	}
}

func make_faas_req4(url string, in interface{}, succ_code int, tmo uint) (*http.Response, error) {
	address := gateProto() + "://" + conf.Login.Host + ":" + conf.Login.Port + "/v1/" + url

	h := make(map[string]string)
	if conf.Login.Token != "" {
		h["X-Auth-Token"] = conf.Login.Token
	}

	var crt []byte
	if conf.TLS && conf.Certs != "" {
		var err error

		crt, err = ioutil.ReadFile(conf.Certs)
		if err != nil {
			return nil, fmt.Errorf("Error reading cert file: %s", err.Error())
		}
	}

	return swyhttp.MarshalAndPost(
			&swyhttp.RestReq{
				Address:	address,
				Headers:	h,
				Success:	succ_code,
				Timeout:	tmo,
				Certs:		crt,
			}, in)
}

func faas_login() string {
	resp, err := make_faas_req4("login", swyapi.UserLogin {
			UserName: conf.Login.User, Password: conf.Login.Pass,
		}, http.StatusOK, 0)
	if err != nil {
		fatal(err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fatal(fmt.Errorf("Bad responce from server: " + string(resp.Status)))
	}

	token := resp.Header.Get("X-Subject-Token")
	if token == "" {
		fatal(fmt.Errorf("No auth token from server"))
	}

	var td swyapi.UserToken
	err = swyhttp.ReadAndUnmarshalResp(resp, &td)
	if err != nil {
		fatal(fmt.Errorf("Can't unmarshal login resp: %s", err.Error()))
	}

	fmt.Printf("Logged in, token till %s\n", td.Expires)

	return token
}

func make_faas_req3(url string, in interface{}, succ_code int, tmo uint) *http.Response {
	first_attempt := true
again:
	resp, err := make_faas_req4(url, in, succ_code, tmo)
	if err != nil {
		if resp == nil {
			fatal(err)
		}

		if (resp.StatusCode == http.StatusUnauthorized) && first_attempt {
			resp.Body.Close()
			first_attempt = false
			refresh_token("")
			goto again
		}

		if resp.StatusCode == http.StatusBadRequest {
			var gerr swyapi.GateErr

			err = swyhttp.ReadAndUnmarshalResp(resp, &gerr)
			resp.Body.Close()

			if err == nil {
				err = fmt.Errorf("Operation failed (%d): %s", gerr.Code, gerr.Message)
			} else {
				err = fmt.Errorf("Operation failed with no details")
			}
		} else {
			err = fmt.Errorf("Bad responce: %s", string(resp.Status))
		}

		fatal(err)
	}

	return resp
}

func make_faas_req2(url string, in interface{}, out interface{}, succ_code int, tmo uint) {
	resp := make_faas_req3(url, in, succ_code, tmo)
	/* Here we have http.StatusOK */
	defer resp.Body.Close()

	if out != nil {
		err := swyhttp.ReadAndUnmarshalResp(resp, out)
		if err != nil {
			fatal(err)
		}
	}
}

func make_faas_req(url string, in interface{}, out interface{}) {
	make_faas_req2(url, in, out, http.StatusOK, 30)
}

func list_users(args []string, opts [16]string) {
	var uss []swyapi.UserInfo
	make_faas_req("users", swyapi.ListUsers{}, &uss)

	for _, u := range uss {
		fmt.Printf("%s (%s)\n", u.Id, u.Name)
	}
}

func add_user(args []string, opts [16]string) {
	make_faas_req2("adduser", swyapi.AddUser{Id: args[0], Pass: opts[1], Name: opts[0]},
		nil, http.StatusCreated, 0)
}

func del_user(args []string, opts [16]string) {
	make_faas_req2("deluser", swyapi.UserInfo{Id: args[0]}, nil, http.StatusNoContent, 0)
}

func set_password(args []string, opts [16]string) {
	make_faas_req2("setpass", swyapi.UserLogin{UserName: args[0], Password: opts[0]},
		nil, http.StatusCreated, 0)
}

func show_user_info(args []string, opts [16]string) {
	var ui swyapi.UserInfo
	make_faas_req("userinfo", swyapi.UserInfo{Id: args[0]}, &ui)
	fmt.Printf("Name: %s\n", ui.Name)
}

func do_user_limits(args []string, opts [16]string) {
	var l swyapi.UserLimits
	chg := false

	if opts[0] != "" {
		l.PlanId = opts[0]
		chg = true
	}

	if opts[1] != "" {
		if l.Fn == nil {
			l.Fn = &swyapi.FunctionLimits{}
		}
		l.Fn.Rate, l.Fn.Burst = parse_rate(opts[1])
		chg = true
	}

	if opts[2] != "" {
		if l.Fn == nil {
			l.Fn = &swyapi.FunctionLimits{}
		}
		v, err := strconv.ParseUint(opts[2], 10, 32)
		if err != nil {
			fatal(fmt.Errorf("Bad max-fn value %s: %s", opts[2], err.Error()))
		}
		l.Fn.MaxInProj = uint(v)
		chg = true
	}

	if opts[3] != "" {
		if l.Fn == nil {
			l.Fn = &swyapi.FunctionLimits{}
		}
		v, err := strconv.ParseFloat(opts[3], 64)
		if err != nil {
			fatal(fmt.Errorf("Bad GBS value %s: %s", opts[3], err.Error()))
		}
		l.Fn.GBS = v
		chg = true
	}

	if opts[4] != "" {
		if l.Fn == nil {
			l.Fn = &swyapi.FunctionLimits{}
		}
		v, err := strconv.ParseUint(opts[4], 10, 64)
		if err != nil {
			fatal(fmt.Errorf("Bad bytes-out value %s: %s", opts[4], err.Error()))
		}
		l.Fn.BytesOut = v
		chg = true
	}

	if chg {
		l.Id = args[0]
		make_faas_req("limits/set", &l, nil)
	} else {
		make_faas_req("limits/get", swyapi.UserInfo{Id: args[0]}, &l)
		if l.PlanId != "" {
			fmt.Printf("Plan ID: %s\n", l.PlanId)
		}
		if l.Fn != nil {
			fmt.Printf("Functions:\n")
			if l.Fn.Rate != 0 {
				fmt.Printf("    Rate:              %d:%d\n", l.Fn.Rate, l.Fn.Burst)
			}
			if l.Fn.MaxInProj != 0 {
				fmt.Printf("    Max in project:    %d\n", l.Fn.MaxInProj)
			}
			if l.Fn.GBS != 0 {
				fmt.Printf("    Max GBS:           %f\n", l.Fn.GBS)
			}
			if l.Fn.BytesOut != 0 {
				fmt.Printf("    Max bytes out:     %d\n", formatBytes(l.Fn.BytesOut))
			}
		}
	}
}

func dateOnly(tm string) string {
	if tm == "" {
		return ""
	}

	t, _ := time.Parse(time.RFC1123Z, tm)
	return t.Format("02 Jan 2006")
}

func show_stats(args []string, opts [16]string) {
	var rq swyapi.TenantStatsReq
	var st swyapi.TenantStatsResp
	var err error

	if opts[0] != "" {
		rq.Periods, err = strconv.Atoi(opts[0])
		if err != nil {
			fatal(fmt.Errorf("Bad period value %s: %s", opts[0],  err.Error()))
		}
	}

	make_faas_req("stats", rq, &st)

	for _, s := range(st.Stats) {
		fmt.Printf("---\n%s ... %s\n", dateOnly(s.From), dateOnly(s.Till))
		fmt.Printf("Called:           %d\n", s.Called)
		fmt.Printf("GBS:              %f\n", s.GBS)
		fmt.Printf("Bytes sent:       %s\n", formatBytes(s.BytesOut))
	}
}

func list_projects(args []string, opts [16]string) {
	var ps []swyapi.ProjectItem
	make_faas_req("project/list", swyapi.ProjectList{}, &ps)

	for _, p := range ps {
		fmt.Printf("%s\n", p.Project)
	}
}

func list_functions(project string, args []string, opts [16]string) {
	if opts[0] == "" {
		var fns []swyapi.FunctionItem
		make_faas_req("function/list", swyapi.FunctionList{ Project: project, }, &fns)

		fmt.Printf("%-20s%-10s\n", "NAME", "STATE")
		for _, fn := range fns {
			fmt.Printf("%-20s%-12s\n", fn.FuncName, fn.State)
		}
	} else if opts[0] == "json" {
		resp := make_faas_req3("function/list/info",
			swyapi.FunctionListInfo{ Project: project, Periods: 0 }, http.StatusOK, 30)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fatal(fmt.Errorf("\tCan't parse responce: %s", err.Error()))
		}
		fmt.Printf("%s\n", string(body))
	} else {
		fatal(fmt.Errorf("Bad -o value %s", opts[0]))
	}
}

func sb2s(b uint64, o uint64, s string) string {
	if b >= 1 << o {
		i := b >> o
		r := ((b - (i<<o)) >> (o-10))
		if r >= 1000 {
			/* Values 1000 through 1023 are valid, but should
			 * not result in .10 fraction below. Maybe we should
			 * rather split tenths by 102? Maybe 103? Dunno.
			 */
			r = 999
		}

		if r >= 100 {
			return fmt.Sprintf("%d.%d %s", i, r/100, s)
		} else {
			return fmt.Sprintf("%d %s", i, s)
		}
	}
	return ""
}

func formatBytes(b uint64) string {
	var bo string

	bo = sb2s(b, 30, "Gb")
	if bo == "" {
		bo = sb2s(b, 20, "Mb")
		if bo == "" {
			bo = sb2s(b, 10, "Kb")
			if bo == "" {
				bo = fmt.Sprintf("%d bytes", b)
			}
		}
	}

	return bo
}

func info_function(project string, args []string, opts [16]string) {
	var ifo swyapi.FunctionInfo
	make_faas_req("function/info", swyapi.FunctionInfoReq{ Project: project, FuncName: args[0], Periods: 0}, &ifo)
	ver := ifo.Version
	if len(ver) > 8 {
		ver = ver[:8]
	}

	fmt.Printf("Lang:        %s\n", ifo.Code.Lang)

	rv := ""
	if len(ifo.RdyVersions) != 0 {
		rv = " (" + strings.Join(ifo.RdyVersions, ",") + ")"
	}
	fmt.Printf("Version:     %s%s\n", ver, rv)
	fmt.Printf("State:       %s\n", ifo.State)
	if len(ifo.Mware) > 0 {
		fmt.Printf("Mware:       %s\n", strings.Join(ifo.Mware, ", "))
	}
	if ifo.Event.Source != "" {
		estr := ifo.Event.Source
		if ifo.Event.Source == "url" {
			/* nothing */
		} else if ifo.Event.Source == "cron" {
			for _, c := range(ifo.Event.Cron) {
				estr += ":" + c.Tab + "/" + make_args_string(c.Args)
			}
		} else {
			estr += "UNKNOWN"
		}
		fmt.Printf("Event:       %s\n", estr)
	}
	if ifo.URL != "" {
		fmt.Printf("URL:         %s://%s\n", gateProto(), ifo.URL)
	}
	fmt.Printf("Timeout:     %dms\n", ifo.Size.Timeout)
	if ifo.Size.Rate != 0 {
		fmt.Printf("Rate:        %d:%d\n", ifo.Size.Rate, ifo.Size.Burst)
	}
	fmt.Printf("Memory:      %dMi\n", ifo.Size.Memory)
	fmt.Printf("Called:      %d\n", ifo.Stats[0].Called)
	if ifo.Stats[0].Called != 0 {
		lc, _ := time.Parse(time.RFC1123Z, ifo.Stats[0].LastCall)
		since := time.Since(lc)
		since -= since % time.Second
		fmt.Printf("Last run:    %s ago\n", since.String())
		fmt.Printf("Time:        %d (avg %d) usec\n", ifo.Stats[0].Time, ifo.Stats[0].Time / ifo.Stats[0].Called)
		fmt.Printf("GBS:         %f\n", ifo.Stats[0].GBS)
	}

	if b := ifo.Stats[0].BytesOut; b != 0 {
		fmt.Printf("Bytes sent:  %s\n", formatBytes(b))
	}

	if ifo.AuthCtx != "" {
		fmt.Printf("Auth by:     %s\n", ifo.AuthCtx)
	}

	if ifo.UserData != "" {
		fmt.Printf("Data:        %s\n", ifo.UserData)
	}
}

func check_lang(args []string, opts [16]string) {
	l := detect_language(opts[0], "code")
	fmt.Printf("%s\n", l)
}

func detect_language(path string, typ string) string {
	if typ != "code" {
		fatal(fmt.Errorf("can't detect repo language"))
		return ""
	}

	cont, err := ioutil.ReadFile(path)
	if err != nil {
		fatal(fmt.Errorf("Can't read sources: %s", err.Error()))
		return ""
	}

	pyr := regexp.MustCompile("^def\\s+main\\s*\\(")
	gor := regexp.MustCompile("^func\\s+Main\\s*\\(.*interface\\s*{\\s*}")
	swr := regexp.MustCompile("^func\\s+Main\\s*\\(.*->\\s*Encodable")

	lines := strings.Split(string(cont), "\n")
	for _, ln := range(lines) {
		if pyr.MatchString(ln) {
			return "python"
		}

		if gor.MatchString(ln) {
			return "golang"
		}

		if swr.MatchString(ln) {
			return "swift"
		}
	}

	return ""
}

func encodeFile(file string) string {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		fatal(fmt.Errorf("Can't read file sources: %s", err.Error()))
	}

	return base64.StdEncoding.EncodeToString(data)
}

func parse_rate(val string) (uint, uint) {
	var rate, burst uint64
	var err error

	rl := strings.Split(val, ":")
	rate, err = strconv.ParseUint(rl[0], 10, 32)
	if err != nil {
		fatal(fmt.Errorf("Bad rate value %s: %s", rl[0], err.Error()))
	}
	if len(rl) > 1 {
		burst, err = strconv.ParseUint(rl[1], 10, 32)
		if err != nil {
			fatal(fmt.Errorf("Bad burst value %s: %s", rl[1], err.Error()))
		}
	}

	return uint(rate), uint(burst)
}

func add_function(project string, args []string, opts [16]string) {
	sources := swyapi.FunctionSources{}
	code := swyapi.FunctionCode{}

	st, err := os.Stat(opts[1])
	if err != nil {
		fatal(fmt.Errorf("Can't stat sources path"))
	}

	if st.IsDir() {
		repo, err := filepath.Abs(opts[1])
		if err != nil {
			fatal(fmt.Errorf("Can't get abs path for repo"))
		}

		fmt.Printf("Will add git repo %s\n", repo)
		sources.Type = "git"
		sources.Repo = repo
	} else {
		fmt.Printf("Will add file %s\n", opts[1])
		sources.Type = "code"
		sources.Code = encodeFile(opts[1])
	}

	if opts[0] == "" {
		opts[0] = detect_language(opts[1], sources.Type)
		fmt.Printf("Detected lang to %s\n", opts[0])
	}

	code.Lang = opts[0]
	if opts[7] != "" {
		code.Env = strings.Split(opts[7], ":")
	}

	var mw []string
	if opts[2] != "" {
		mw = strings.Split(opts[2], ",")
	}

	evt := swyapi.FunctionEvent {}
	if opts[3] != "" {
		mwe := strings.Split(opts[3], ":")
		evt.Source = mwe[0]
		if evt.Source == "url" {
			/* nothing */
		} else if evt.Source == "cron" {
			for _, cron := range mwe[1:] {
				x := strings.SplitN(cron, "/", 2)
				evt.Cron = append(evt.Cron, swyapi.FunctionEventCron {
					Tab: x[0],
					Args: split_args_string(x[1]),
				})
			}
		} else {
			/* FIXME -- CRONTAB */
			fatal(fmt.Errorf("Unknown event string"))
		}
	}

	req := swyapi.FunctionAdd{
		Project: project,
		FuncName: args[0],
		Sources: sources,
		Code: code,
		Mware: mw,
		Event: evt,
	}

	if opts[4] != "" {
		req.Size.Timeout, err = strconv.ParseUint(opts[4], 10, 64)
		if err != nil {
			fatal(fmt.Errorf("Bad tmo value %s: %s", opts[4], err.Error()))
		}
	}

	if opts[5] != "" {
		req.Size.Rate, req.Size.Burst = parse_rate(opts[5])
	}

	if opts[6] != "" {
		req.UserData = opts[6]
	}

	if opts[8] != "" {
		req.AuthCtx = opts[8]
	}

	make_faas_req("function/add", req, nil)

}

/* Splits a=v,a=v,... string into map */
func split_args_string(as string) map[string]string {
	ret := make(map[string]string)
	for _, arg := range strings.Split(as, ",") {
		a := strings.SplitN(arg, "=", 2)
		ret[a[0]] = a[1]
	}
	return ret
}

func make_args_string(args map[string]string) string {
	var ass []string
	for a, v := range args {
		ass = append(ass, a + "=" + v)
	}
	return strings.Join(ass, ",")
}

func run_function(project string, args []string, opts [16]string) {
	var rres swyapi.FunctionRunResult

	argmap := split_args_string(args[1])
	make_faas_req("function/run",
		swyapi.FunctionRun{ Project: project, FuncName: args[0], Args: argmap, }, &rres)

	fmt.Printf("returned: %s\n", rres.Return)
	fmt.Printf("%s", rres.Stdout)
	fmt.Fprintf(os.Stderr, "%s", rres.Stderr)
}

func update_function(project string, args []string, opts [16]string) {
	req := swyapi.FunctionUpdate {
		Project: project,
		FuncName: args[0],
	}

	if opts[0] != "" {
		req.Code = encodeFile(opts[0])
	}

	if opts[1] != "" {
		var err error
		req.Size = &swyapi.FunctionSize {}
		req.Size.Timeout, err = strconv.ParseUint(opts[1], 10, 64)
		if err != nil {
			fatal(fmt.Errorf("Bad tmo value %s: %s", opts[4], err.Error()))
		}
	}

	if opts[2] != "" {
		if req.Size == nil {
			req.Size = &swyapi.FunctionSize {}
		}

		req.Size.Rate, req.Size.Burst = parse_rate(opts[2])
	}

	if opts[3] != "" {
		if opts[3] == "-" {
			req.Mware = &[]string{}
		} else {
			mw := strings.Split(opts[3], ",")
			req.Mware = &mw
		}
	}

	if opts[4] != "" {
		req.UserData = opts[4]
	}

	if opts[7] != "" {
		var ac string
		if opts[7] != "-" {
			ac = opts[7]
		}

		req.AuthCtx = &ac
	}

	make_faas_req("function/update", req, nil)

	if opts[5] != "" {
		fmt.Printf("Wait FN %s\n", opts[5])
		wait_fn(project, []string{args[0]}, [16]string{opts[5], "15000"})
	}

	if opts[6] != "" {
		fmt.Printf("Run FN %s\n", opts[6])
		run_function(project, []string{args[0], opts[6]}, [16]string{})
	}
}

func del_function(project string, args []string, opts [16]string) {
	make_faas_req("function/remove",
		swyapi.FunctionRemove{ Project: project, FuncName: args[0] }, nil)
}

func on_fn(project string, args []string, opts [16]string) {
	make_faas_req("function/state",
		swyapi.FunctionState{ Project: project, FuncName: args[0], State: "ready" }, nil)
}

func off_fn(project string, args []string, opts [16]string) {
	make_faas_req("function/state",
		swyapi.FunctionState{ Project: project, FuncName: args[0], State: "deactivated" }, nil)
}

func wait_fn(project string, args []string, opts [16]string) {
	req := swyapi.FunctionWait{
		Project: project,
		FuncName: args[0],
	}

	if opts[0] != "" {
		req.Version = opts[0]
	}

	if opts[1] != "" {
		tmo, err := strconv.ParseUint(opts[1], 10, 32)
		if err != nil {
			fatal(fmt.Errorf("Bad timeout value"))
		}

		req.Timeout = uint32(tmo)
	}

	var httpTmo uint /* seconds */
	httpTmo = uint((time.Duration(req.Timeout) * time.Millisecond) / time.Second)
	if httpTmo <= 1 {
		httpTmo = 1
	}

	make_faas_req2("function/wait", req, nil, http.StatusOK, httpTmo + 10)
}

func show_code(project string, args []string, opts [16]string) {
	var res swyapi.FunctionSources
	make_faas_req("function/code",
		swyapi.FunctionXID{ Project: project, FuncName: args[0], Version: opts[0], }, &res)

	data, err := base64.StdEncoding.DecodeString(res.Code)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("%s", data)
}

func show_logs(project string, args []string, opts [16]string) {
	var res []swyapi.FunctionLogEntry
	make_faas_req("function/logs",
		swyapi.FunctionID{ Project: project, FuncName: args[0], }, &res)

	for _, le := range res {
		fmt.Printf("%36s%8s: %s\n", le.Ts, le.Event, le.Text)
	}
}

func list_mware(project string, args []string, opts [16]string) {
	req := swyapi.MwareList{ Project: project }
	if opts[1] != "" {
		req.Type = opts[1]
	}

	if opts[0] == "" {
		var mws []swyapi.MwareItem
		make_faas_req("mware/list", &req, &mws)

		fmt.Printf("%-20s%-10s\n", "NAME", "TYPE")
		for _, mw := range mws {
			fmt.Printf("%-20s%-10s%s\n", mw.ID, mw.Type, "")
		}
	} else if opts[0] == "json" {
		resp := make_faas_req3("mware/list/info", &req, http.StatusOK, 30)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fatal(fmt.Errorf("\tCan't parse responce: %s", err.Error()))
		}
		fmt.Printf("%s\n", string(body))
	} else {
		fatal(fmt.Errorf("Bad -o value %s", opts[0]))
	}
}

func info_mware(project string, args []string, opts [16]string) {
	var resp swyapi.MwareInfo

	make_faas_req("mware/info",
		swyapi.MwareID{ Project: project, ID: args[0], }, &resp)
	fmt.Printf("Type:         %s\n", resp.Type)
	if resp.DU != nil {
		fmt.Printf("Disk usage:   %s\n", formatBytes(*resp.DU << 10))
	}
	if resp.UserData != "" {
		fmt.Printf("Data:         %s\n", resp.UserData)
	}
}

func add_mware(project string, args []string, opts [16]string) {
	req := swyapi.MwareAdd {
		Project: project,
		ID: args[0],
		Type: args[1],
		UserData: opts[0],
	}
	make_faas_req("mware/add", req, nil)
}

func del_mware(project string, args []string, opts [16]string) {
	make_faas_req("mware/remove",
		swyapi.MwareRemove{ Project: project, ID: args[0], }, nil)
}

func s3_access(project string, args []string, opts [16]string) {
	lt, err := strconv.Atoi(opts[0])
	if err != nil {
		fatal(fmt.Errorf("Bad lifetie value: %s", err.Error()))
	}

	var creds swyapi.MwareS3Creds

	make_faas_req("mware/access/s3",
		swyapi.MwareS3Access{ Project: project, Bucket: args[0], Lifetime: uint32(lt)}, &creds)
	fmt.Printf("Endpoint %s\n", creds.Endpoint)
	fmt.Printf("Key:     %s\n", creds.Key)
	fmt.Printf("Secret:  %s\n", creds.Secret)
	fmt.Printf("Expires: in %d seconds\n", creds.Expires)
}

func req_list(url string) {
	var r []string

	make_faas_req(url, nil, &r)
	for _, v := range r {
		fmt.Printf("%s\n", v)
	}
}

func languages(args []string, opts [16]string) {
	req_list("info/langs")
}

func mware_types(args []string, opts [16]string) {
	req_list("info/mwares")
}

func login() {
	home, found := os.LookupEnv("HOME")
	if !found {
		fatal(fmt.Errorf("No HOME dir set"))
	}

	err := swy.ReadYamlConfig(home + "/.swifty.conf", &conf)
	if err != nil {
		fatal(fmt.Errorf("Login first"))
	}
}

func show_login() {
	fmt.Printf("%s@%s:%s (%s)\n", conf.Login.User, conf.Login.Host, conf.Login.Port, gateProto())
}

func manage_login(args []string, opts [16]string) {
	action := "show"
	if len(args) >= 1 {
		action = args[0]
	}

	switch action {
	case "show":
		show_login()
	}
}

func make_login(creds string, opts [16]string) {
	//
	// Login string is user:pass@host:port
	//
	// swifty.user:swifty@10.94.96.216:8686
	//
	home, found := os.LookupEnv("HOME")
	if !found {
		fatal(fmt.Errorf("No HOME dir set"))
	}

	c := swy.ParseXCreds(creds)
	conf.Login.User = c.User
	conf.Login.Pass = c.Pass
	conf.Login.Host = c.Host
	conf.Login.Port = c.Port

	if opts[0] == "yes" {
		conf.TLS = true
		if opts[1] != "" {
			conf.Certs = opts[1]
		}
	}

	refresh_token(home)
}

func refresh_token(home string) {
	if home == "" {
		var found bool
		home, found = os.LookupEnv("HOME")
		if !found {
			fatal(fmt.Errorf("No HOME dir set"))
		}
	}

	conf.Login.Token = faas_login()

	err := swy.WriteYamlConfig(home + "/.swifty.conf", &conf)
	if err != nil {
		fatal(fmt.Errorf("Can't write swifty.conf: %s", err.Error()))
	}
}

const (
	CMD_LOGIN string	= "login"
	CMD_ME string		= "me"
	CMD_STATS string	= "st"
	CMD_PS string		= "ps"
	CMD_LS string		= "ls"
	CMD_INF string		= "inf"
	CMD_ADD string		= "add"
	CMD_RUN string		= "run"
	CMD_UPD string		= "upd"
	CMD_DEL string		= "del"
	CMD_LOGS string		= "logs"
	CMD_CODE string		= "code"
	CMD_ON string		= "on"
	CMD_OFF string		= "off"
	CMD_WAIT string		= "wait"
	CMD_MLS string		= "mls"
	CMD_MINF string		= "minf"
	CMD_MADD string		= "madd"
	CMD_MDEL string		= "mdel"
	CMD_S3ACC string	= "s3acc"
	CMD_LUSR string		= "uls"
	CMD_UADD string		= "uadd"
	CMD_UDEL string		= "udel"
	CMD_PASS string		= "pass"
	CMD_UINF string		= "uinf"
	CMD_LIMITS string	= "limits"
	CMD_MTYPES string	= "mt"
	CMD_LANGS string	= "lng"
	CMD_LANG string		= "ld"
)

var cmdOrder = []string {
	CMD_LOGIN,
	CMD_ME,
	CMD_STATS,
	CMD_PS,
	CMD_LS,
	CMD_INF,
	CMD_ADD,
	CMD_RUN,
	CMD_UPD,
	CMD_DEL,
	CMD_LOGS,
	CMD_CODE,
	CMD_ON,
	CMD_OFF,
	CMD_WAIT,
	CMD_MLS,
	CMD_MINF,
	CMD_MADD,
	CMD_MDEL,
	CMD_S3ACC,
	CMD_LUSR,
	CMD_UADD,
	CMD_UDEL,
	CMD_PASS,
	CMD_UINF,
	CMD_LIMITS,
	CMD_LANGS,
	CMD_MTYPES,
	CMD_LANG,
}

type cmdDesc struct {
	opts	*flag.FlagSet
	pargs	[]string
	project	string
	pcall	func(string, []string, [16]string)
	call	func([]string, [16]string)
}

var cmdMap = map[string]*cmdDesc {
	CMD_LOGIN:	&cmdDesc{			  opts: flag.NewFlagSet(CMD_LOGIN, flag.ExitOnError) },
	CMD_ME:		&cmdDesc{  call: manage_login,	  opts: flag.NewFlagSet(CMD_ME, flag.ExitOnError) },
	CMD_STATS:	&cmdDesc{  call: show_stats,	  opts: flag.NewFlagSet(CMD_STATS, flag.ExitOnError) },
	CMD_PS:		&cmdDesc{  call: list_projects,	  opts: flag.NewFlagSet(CMD_PS, flag.ExitOnError) },
	CMD_LS:		&cmdDesc{ pcall: list_functions,  opts: flag.NewFlagSet(CMD_LS, flag.ExitOnError) },
	CMD_INF:	&cmdDesc{ pcall: info_function,	  opts: flag.NewFlagSet(CMD_INF, flag.ExitOnError) },
	CMD_ADD:	&cmdDesc{ pcall: add_function,	  opts: flag.NewFlagSet(CMD_ADD, flag.ExitOnError) },
	CMD_RUN:	&cmdDesc{ pcall: run_function,	  opts: flag.NewFlagSet(CMD_RUN, flag.ExitOnError) },
	CMD_UPD:	&cmdDesc{ pcall: update_function, opts: flag.NewFlagSet(CMD_UPD, flag.ExitOnError) },
	CMD_DEL:	&cmdDesc{ pcall: del_function,	  opts: flag.NewFlagSet(CMD_DEL, flag.ExitOnError) },
	CMD_LOGS:	&cmdDesc{ pcall: show_logs,	  opts: flag.NewFlagSet(CMD_LOGS, flag.ExitOnError) },
	CMD_CODE:	&cmdDesc{ pcall: show_code,	  opts: flag.NewFlagSet(CMD_CODE, flag.ExitOnError) },
	CMD_ON:		&cmdDesc{ pcall: on_fn,		  opts: flag.NewFlagSet(CMD_ON, flag.ExitOnError) },
	CMD_OFF:	&cmdDesc{ pcall: off_fn,	  opts: flag.NewFlagSet(CMD_OFF, flag.ExitOnError) },
	CMD_WAIT:	&cmdDesc{ pcall: wait_fn,	  opts: flag.NewFlagSet(CMD_WAIT, flag.ExitOnError) },
	CMD_MLS:	&cmdDesc{ pcall: list_mware,	  opts: flag.NewFlagSet(CMD_MLS, flag.ExitOnError) },
	CMD_MINF:	&cmdDesc{ pcall: info_mware,	  opts: flag.NewFlagSet(CMD_INF, flag.ExitOnError) },
	CMD_MADD:	&cmdDesc{ pcall: add_mware,	  opts: flag.NewFlagSet(CMD_MADD, flag.ExitOnError) },
	CMD_MDEL:	&cmdDesc{ pcall: del_mware,	  opts: flag.NewFlagSet(CMD_MDEL, flag.ExitOnError) },
	CMD_S3ACC:	&cmdDesc{ pcall: s3_access,	  opts: flag.NewFlagSet(CMD_S3ACC, flag.ExitOnError) },
	CMD_LUSR:	&cmdDesc{  call: list_users,	  opts: flag.NewFlagSet(CMD_LUSR, flag.ExitOnError) },
	CMD_UADD:	&cmdDesc{  call: add_user,	  opts: flag.NewFlagSet(CMD_UADD, flag.ExitOnError) },
	CMD_UDEL:	&cmdDesc{  call: del_user,	  opts: flag.NewFlagSet(CMD_UDEL, flag.ExitOnError) },
	CMD_PASS:	&cmdDesc{  call: set_password,	  opts: flag.NewFlagSet(CMD_PASS, flag.ExitOnError) },
	CMD_UINF:	&cmdDesc{  call: show_user_info,  opts: flag.NewFlagSet(CMD_UINF, flag.ExitOnError) },
	CMD_LIMITS:	&cmdDesc{  call: do_user_limits,  opts: flag.NewFlagSet(CMD_LIMITS, flag.ExitOnError) },
	CMD_LANGS:	&cmdDesc{  call: languages,	  opts: flag.NewFlagSet(CMD_LANGS, flag.ExitOnError) },
	CMD_MTYPES:	&cmdDesc{  call: mware_types,	  opts: flag.NewFlagSet(CMD_MTYPES, flag.ExitOnError) },
	CMD_LANG:	&cmdDesc{  call: check_lang,	  opts: flag.NewFlagSet(CMD_LANG, flag.ExitOnError) },
}

func bindCmdUsage(cmd string, args []string, help string, wp bool) {
	cd := cmdMap[cmd]
	if wp {
		cd.opts.StringVar(&cd.project, "proj", "", "Project to work on")
	}

	cd.pargs = args
	cd.opts.Usage = func() {
		var astr string
		if len(args) != 0 {
			astr = " <" + strings.Join(args, "> <") + ">"
		}
		fmt.Fprintf(os.Stderr, "%s%s\n\t%s\n", cmd, astr, help)
		cd.opts.PrintDefaults()
	}
}

func main() {
	var opts [16]string

	cmdMap[CMD_LOGIN].opts.StringVar(&opts[0], "tls", "no", "TLS mode")
	cmdMap[CMD_LOGIN].opts.StringVar(&opts[1], "cert", "", "x509 cert file")
	bindCmdUsage(CMD_LOGIN,	[]string{"USER:PASS@HOST:PORT"}, "Login into the system", false)

	bindCmdUsage(CMD_ME, []string{"ACTION"}, "Manage login", false)

	cmdMap[CMD_STATS].opts.StringVar(&opts[0], "p", "0", "Periods to report")
	bindCmdUsage(CMD_STATS,	[]string{}, "Show stats", false)
	bindCmdUsage(CMD_PS,	[]string{}, "List projects", false)

	cmdMap[CMD_LS].opts.StringVar(&opts[0], "o", "", "Output format (NONE, json)")
	bindCmdUsage(CMD_LS,	[]string{}, "List functions", true)
	bindCmdUsage(CMD_INF,	[]string{"NAME"}, "Function info", true)
	cmdMap[CMD_ADD].opts.StringVar(&opts[0], "lang", "auto", "Language")
	cmdMap[CMD_ADD].opts.StringVar(&opts[1], "src", ".", "Source file")
	cmdMap[CMD_ADD].opts.StringVar(&opts[2], "mw", "", "Mware to use, comma-separated")
	cmdMap[CMD_ADD].opts.StringVar(&opts[3], "event", "", "Event this fn is to start")
	cmdMap[CMD_ADD].opts.StringVar(&opts[4], "tmo", "", "Timeout")
	cmdMap[CMD_ADD].opts.StringVar(&opts[5], "rl", "", "Rate (rate[:burst])")
	cmdMap[CMD_ADD].opts.StringVar(&opts[6], "data", "", "Any text associated with fn")
	cmdMap[CMD_ADD].opts.StringVar(&opts[7], "env", "", "Colon-separated list of env vars")
	cmdMap[CMD_ADD].opts.StringVar(&opts[8], "auth", "", "ID of auth mware to verify the call")
	bindCmdUsage(CMD_ADD,	[]string{"NAME"}, "Add a function", true)
	bindCmdUsage(CMD_RUN,	[]string{"NAME", "ARG=VAL,..."}, "Run a function", true)
	cmdMap[CMD_UPD].opts.StringVar(&opts[0], "src", "", "Source file")
	cmdMap[CMD_UPD].opts.StringVar(&opts[1], "tmo", "", "Timeout")
	cmdMap[CMD_UPD].opts.StringVar(&opts[2], "rl", "", "Rate (rate[:burst])")
	cmdMap[CMD_UPD].opts.StringVar(&opts[3], "mw", "", "Mware to use, comma-separated")
	cmdMap[CMD_UPD].opts.StringVar(&opts[4], "data", "", "Associated text")
	cmdMap[CMD_UPD].opts.StringVar(&opts[5], "ver", "", "Version")
	cmdMap[CMD_UPD].opts.StringVar(&opts[6], "arg", "", "Args")
	cmdMap[CMD_UPD].opts.StringVar(&opts[7], "auth", "", "Auth context (- for off)")
	bindCmdUsage(CMD_UPD,	[]string{"NAME"}, "Update a function", true)
	bindCmdUsage(CMD_DEL,	[]string{"NAME"}, "Delete a function", true)
	bindCmdUsage(CMD_LOGS,	[]string{"NAME"}, "Show function logs", true)
	cmdMap[CMD_CODE].opts.StringVar(&opts[0], "version", "", "Version")
	bindCmdUsage(CMD_CODE,  []string{"NAME"}, "Show function code", true)
	bindCmdUsage(CMD_ON,	[]string{"NAME"}, "Activate function", true)
	bindCmdUsage(CMD_OFF,	[]string{"NAME"}, "Deactivate function", true)

	cmdMap[CMD_WAIT].opts.StringVar(&opts[0], "version", "", "Version")
	cmdMap[CMD_WAIT].opts.StringVar(&opts[1], "tmo", "", "Timeout")
	bindCmdUsage(CMD_WAIT,	[]string{"NAME"}, "Wait function event", true)

	cmdMap[CMD_MLS].opts.StringVar(&opts[0], "o", "", "Output format (NONE, json)")
	cmdMap[CMD_MLS].opts.StringVar(&opts[1], "type", "", "Filter mware by type")
	bindCmdUsage(CMD_MLS,	[]string{}, "List middleware", true)
	bindCmdUsage(CMD_MINF,	[]string{"ID"}, "Middleware info", true)
	cmdMap[CMD_MADD].opts.StringVar(&opts[0], "data", "", "Associated text")
	bindCmdUsage(CMD_MADD,	[]string{"ID", "TYPE"}, "Add middleware", true)
	bindCmdUsage(CMD_MDEL,	[]string{"ID"}, "Delete middleware", true)

	cmdMap[CMD_S3ACC].opts.StringVar(&opts[0], "life", "60", "Lifetime (default 1 min)")
	bindCmdUsage(CMD_S3ACC,	[]string{"BUCKET"}, "Get keys for S3", true)

	bindCmdUsage(CMD_LUSR,	[]string{}, "List users", false)
	cmdMap[CMD_UADD].opts.StringVar(&opts[0], "name", "", "User name")
	cmdMap[CMD_UADD].opts.StringVar(&opts[1], "pass", "", "User password")
	bindCmdUsage(CMD_UADD,	[]string{"UID"}, "Add user", false)
	bindCmdUsage(CMD_UDEL,	[]string{"UID"}, "Del user", false)
	cmdMap[CMD_PASS].opts.StringVar(&opts[0], "pass", "", "New password")
	bindCmdUsage(CMD_PASS,	[]string{"UID"}, "Set password", false)
	bindCmdUsage(CMD_UINF,	[]string{"UID"}, "Get user info", false)
	cmdMap[CMD_LIMITS].opts.StringVar(&opts[0], "plan", "", "Taroff plan ID")
	cmdMap[CMD_LIMITS].opts.StringVar(&opts[1], "rl", "", "Rate (rate[:burst])")
	cmdMap[CMD_LIMITS].opts.StringVar(&opts[2], "fnr", "", "Number of functions (in a project)")
	cmdMap[CMD_LIMITS].opts.StringVar(&opts[3], "gbs", "", "Maximum number of GBS to consume")
	cmdMap[CMD_LIMITS].opts.StringVar(&opts[4], "bo", "", "Maximum outgoing network bytes")
	bindCmdUsage(CMD_LIMITS, []string{"UID"}, "Get/Set limits for user", false)

	bindCmdUsage(CMD_MTYPES, []string{}, "List middleware types", false)
	bindCmdUsage(CMD_LANGS, []string{}, "List of supported languages", false)

	cmdMap[CMD_LANG].opts.StringVar(&opts[0], "src", "", "File")
	bindCmdUsage(CMD_LANG, []string{}, "Check source language", false)

	flag.Usage = func() {
		for _, v := range cmdOrder {
			cmdMap[v].opts.Usage()
		}
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	cd, ok := cmdMap[os.Args[1]]
	if !ok {
		flag.Usage()
		os.Exit(1)
	}

	npa := len(cd.pargs) + 2
	if len(os.Args) >= npa {
		cd.opts.Parse(os.Args[npa:])
	}

	if os.Args[1] == CMD_LOGIN {
		make_login(os.Args[2], opts)
		return
	}

	login()

	if cd.pcall != nil {
		cd.pcall(cd.project, os.Args[2:], opts)
	} else if cd.call != nil {
		cd.call(os.Args[2:], opts)
	} else {
		fatal(fmt.Errorf("Bad cmd"))
	}
}

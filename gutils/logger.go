package gutils

import (
	"errors"
	"fmt"
	"os"
)

//TagTLS -TODO-
const TagTLS = "TLS"

//TagRPCCALL -TODO-
const TagRPCCALL = "RPC.CALL"

//TagRPCERROR -TODO-
const TagRPCERROR = "RPC.ERROR"

//TagUnknown -TODO-
const TagUnknown = "U"

//ErrorRPC -TODO-
type ErrorRPC struct {
	Code    int64
	Message string
}

var enabledTags = map[string]bool{
	//TagTLS: true,
	//TagRPCCALL: true,
	//TagRPCERROR: true,
	TagUnknown: true,
}

func fmtPutLogf(tag, format string, v ...interface{}) {
	if enabledTags[tag] {
		fmt.Printf(tag+" "+format+"\n", v...)
	}
}

//RemoteLogger --TODO--
type RemoteLogger struct {
	tag string
}

//FatalError -TODO-
type FatalError string

func (e FatalError) Error() string {
	return string(e)
}

//NormalError -TODO-
type NormalError string

func (e NormalError) Error() string {
	return string(e)
}

//FormatAll -TODO-
func FormatAll(step string, id int64, debug, format string, v ...interface{}) string {
	var params string

	if len(debug) > 0 {
		params = "[" + debug + "]"
	}

	if len(step) > 0 {
		params = "(" + step + ")" + params
	}

	format = GetCallStack() + params + " " + format

	if id > 0 {
		format = fmt.Sprintf("[%d] ", id) + format
	}

	return fmt.Sprintf(format, v...)
}

//FormatErrorN -TODO-
func FormatErrorN(format string, v ...interface{}) error {
	return NormalError(FormatAll("", 0, "", format, v...))
}

//FormatErrorS -TODO-
func FormatErrorS(step, format string, v ...interface{}) error {
	return NormalError(FormatAll(step, 0, "", format, v...))
}

//FormatErrorI -TODO-
func FormatErrorI(id int64, format string, v ...interface{}) error {
	return NormalError(FormatAll("", id, "", format, v...))
}

//FormatErrorSI -TODO-
func FormatErrorSI(step string, id int64, format string, v ...interface{}) error {
	return NormalError(FormatAll(step, id, "", format, v...))
}

//FormatErrorSD -TODO-
func FormatErrorSD(step, debug, format string, v ...interface{}) error {
	return NormalError(FormatAll(step, 0, debug, format, v...))
}

//FormatFatalF -TODO-
func FormatFatalF(step string, id int64, debug, format string, v ...interface{}) error {
	return FatalError(FormatAll(step, id, debug, format, v...))
}

//FormatFatalS -TODO-
func FormatFatalS(step string, format string, v ...interface{}) error {
	return FatalError(FormatAll(step, 0, "", format, v...))
}

//FormatFatalSI -TODO-
func FormatFatalSI(step string, id int64, format string, v ...interface{}) error {
	return FatalError(FormatAll(step, id, "", format, v...))
}

//FormatFatalSD -TODO-
func FormatFatalSD(step string, debug, format string, v ...interface{}) error {
	return FatalError(FormatAll(step, 0, debug, format, v...))
}

//FormatErrorF -TODO-
func FormatErrorF(step string, id int64, debug, format string, v ...interface{}) error {
	return NormalError(FormatAll(step, id, debug, format, v...))
}

//FormatInfoS -TODO-
func FormatInfoS(step, format string, v ...interface{}) string {
	return FormatAll(step, 0, "", format, v...)
}

//FormatInfoF -TODO-
func FormatInfoF(step string, id int64, format string, v ...interface{}) string {
	return FormatAll(step, id, "", format, v...)
}

//RemoteLog --TODO--
var RemoteLog *RemoteLogger

//InitLog -TODO-
func InitLog(tag string) *RemoteLogger {
	var rl RemoteLogger

	rl.tag = tag

	return &rl
}

//PutDebugN --TODO--
func (rl *RemoteLogger) PutDebugN(format string, v ...interface{}) {
	fmt.Println("DBG " + FormatAll("", 0, "", format, v...))
}

//PutDebugS --TODO--
func (rl *RemoteLogger) PutDebugS(step string, format string, v ...interface{}) {
	fmt.Println("DBG " + FormatAll(step, 0, "", format, v...))
}

//PutDebugI --TODO--
func (rl *RemoteLogger) PutDebugI(id int64, format string, v ...interface{}) {
	fmt.Println("DBG " + FormatAll("", id, "", format, v...))
}

//PutDebugSI --TODO--
func (rl *RemoteLogger) PutDebugSI(step string, id int64, format string, v ...interface{}) {
	fmt.Println("DBG " + FormatAll(step, id, "", format, v...))
}

//PutDebugSD --TODO--
func (rl *RemoteLogger) PutDebugSD(step string, debug string, format string, v ...interface{}) {
	fmt.Println("DBG " + FormatAll(step, 0, debug, format, v...))
}

//PutInfoN --TODO--
func (rl *RemoteLogger) PutInfoN(format string, v ...interface{}) {
	fmt.Println("INF " + FormatAll("", 0, "", format, v...))
}

//PutInfoS --TODO--
func (rl *RemoteLogger) PutInfoS(step string, format string, v ...interface{}) {
	fmt.Println("INF " + FormatAll(step, 0, "", format, v...))
}

//PutInfoI --TODO--
func (rl *RemoteLogger) PutInfoI(id int64, format string, v ...interface{}) {
	fmt.Println("INF " + FormatAll("", id, "", format, v...))
}

//PutInfoSI --TODO--
func (rl *RemoteLogger) PutInfoSI(step string, id int64, format string, v ...interface{}) {
	fmt.Println("INF " + FormatAll(step, id, "", format, v...))
}

//PutInfoSD --TODO--
func (rl *RemoteLogger) PutInfoSD(step string, debug, format string, v ...interface{}) {
	fmt.Println("INF " + FormatAll(step, 0, debug, format, v...))
}

//PutInfoSID --TODO--
func (rl *RemoteLogger) PutInfoSID(step string, id int64, debug, format string, v ...interface{}) {
	fmt.Println("INF " + FormatAll(step, id, debug, format, v...))
}

//PutWarningN --TODO--
func (rl *RemoteLogger) PutWarningN(format string, v ...interface{}) {
	fmt.Println("!WARNING! " + FormatAll("", 0, "", format, v...))
}

//PutWarningS --TODO--
func (rl *RemoteLogger) PutWarningS(step, format string, v ...interface{}) {
	fmt.Println("!WARNING! " + FormatAll(step, 0, "", format, v...))
}

//PutWarningSI --TODO--
func (rl *RemoteLogger) PutWarningSI(step string, id int64, format string, v ...interface{}) {
	fmt.Println("!WARNING! " + FormatAll(step, id, "", format, v...))
}

//PutErrorN --TODO--
func (rl *RemoteLogger) PutErrorN(format string, v ...interface{}) error {
	err := errors.New(FormatAll("", 0, "", format, v...))
	fmt.Println("ERROR " + err.Error())
	return err
}

//PutErrorS --TODO--
func (rl *RemoteLogger) PutErrorS(step, format string, v ...interface{}) error {
	err := errors.New(FormatAll(step, 0, "", format, v...))
	fmt.Println("ERROR " + err.Error())
	return err
}

//PutErrorI --TODO--
func (rl *RemoteLogger) PutErrorI(id int64, format string, v ...interface{}) error {
	err := errors.New(FormatAll("", id, "", format, v...))
	fmt.Println("ERROR " + err.Error())
	return err
}

//PutErrorSI --TODO--
func (rl *RemoteLogger) PutErrorSI(step string, id int64, format string, v ...interface{}) error {
	err := errors.New(FormatAll(step, id, "", format, v...))
	fmt.Println("ERROR " + err.Error())
	return err
}

//PutErrorSD --TODO--
func (rl *RemoteLogger) PutErrorSD(step string, debug string, format string, v ...interface{}) error {
	err := errors.New(FormatAll(step, 0, debug, format, v...))
	fmt.Println("ERROR " + err.Error())
	return err
}

//PutErrorID --TODO--
func (rl *RemoteLogger) PutErrorID(id int64, debug string, format string, v ...interface{}) error {
	err := errors.New(FormatAll("", id, debug, format, v...))
	fmt.Println("ERROR " + err.Error())
	return err
}

//PutErrorSID --TODO--
func (rl *RemoteLogger) PutErrorSID(step string, id int64, debug, format string, v ...interface{}) error {
	err := errors.New(FormatAll(step, id, debug, format, v...))
	fmt.Println("ERROR " + err.Error())
	return err
}

//PutError --TODO--
func (rl *RemoteLogger) PutError(err error) error {
	if fmt.Sprintf("%T", err) == "gutils.FatalError" {
		return rl.PutFatal(err)
	}

	fmt.Println("ERROR " + err.Error())

	return err
}

//PutFatalN --TODO--
func (rl *RemoteLogger) PutFatalN(format string, v ...interface{}) error {
	err := errors.New(FormatAll("", 0, "", format, v...))
	fmt.Println("FATAL " + err.Error())

	os.Exit(1)

	return err
}

//PutFatalS --TODO--
func (rl *RemoteLogger) PutFatalS(step, format string, v ...interface{}) error {
	err := errors.New(FormatAll(step, 0, "", format, v...))
	fmt.Println("FATAL " + err.Error())

	os.Exit(1)

	return err
}

//PutFatalI -TODO-
func (rl *RemoteLogger) PutFatalI(id int64, format string, v ...interface{}) error {
	err := errors.New(FormatAll("", id, "", format, v...))
	fmt.Println("FATAL " + err.Error())

	os.Exit(1)

	return err
}

//PutFatalSI --TODO--
func (rl *RemoteLogger) PutFatalSI(step string, id int64, format string, v ...interface{}) error {
	err := errors.New(FormatAll(step, id, "", format, v...))
	fmt.Println("FATAL " + err.Error())

	os.Exit(1)

	return err
}

//PutFatalID --TODO--
func (rl *RemoteLogger) PutFatalID(id int64, debug string, format string, v ...interface{}) error {
	err := errors.New(FormatAll("", id, debug, format, v...))
	fmt.Println("FATAL " + err.Error())

	os.Exit(1)

	return err
}

//PutFatalSD --TODO--
func (rl *RemoteLogger) PutFatalSD(step string, debug, format string, v ...interface{}) error {
	err := errors.New(FormatAll(step, 0, debug, format, v...))
	fmt.Println("FATAL " + err.Error())

	os.Exit(1)

	return err
}

//PutFatal --TODO--
func (rl *RemoteLogger) PutFatal(err error) error {
	fmt.Println("FATAL " + err.Error())

	os.Exit(1)

	return err
}

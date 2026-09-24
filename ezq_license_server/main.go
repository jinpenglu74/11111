package main

import (
  "crypto/ed25519"
  "crypto/rand"
  "crypto/sha256"
  "encoding/base64"
  "encoding/hex"
  "encoding/json"
  "io"
  "log"
  "net/http"
  "os"
  "strings"
  "sync"
  "time"
)

const product="MistNativeSceneCutter"
const serverVersion="1.4.0-render-free"
const offlineAfter=120*time.Second

type Session struct{ OnlineAt string `json:"onlineAt"`; OfflineAt string `json:"offlineAt,omitempty"`; DurationSeconds int64 `json:"durationSeconds,omitempty"`; DeviceName string `json:"deviceName,omitempty"`; EndReason string `json:"endReason,omitempty"` }
type License struct{
 ID string `json:"id"`; Key string `json:"key"`; Status string `json:"status"`; Note string `json:"note,omitempty"`; CreatedAt string `json:"createdAt"`; DurationDays int `json:"durationDays,omitempty"`; ExpiresAt int64 `json:"expiresAt,omitempty"`; BoundDeviceID string `json:"boundDeviceId,omitempty"`; BoundDeviceName string `json:"boundDeviceName,omitempty"`; ActivatedAt string `json:"activatedAt,omitempty"`; LastSeenAt string `json:"lastSeenAt,omitempty"`; LastOnlineAt string `json:"lastOnlineAt,omitempty"`; LastHeartbeatAt string `json:"lastHeartbeatAt,omitempty"`; LastOfflineAt string `json:"lastOfflineAt,omitempty"`; CurrentOnlineAt string `json:"currentOnlineAt,omitempty"`; OnlineCount int `json:"onlineCount,omitempty"`; Revision int64 `json:"revision,omitempty"`; RevokedDeviceIDs []string `json:"revokedDeviceIds,omitempty"`; Sessions []Session `json:"sessions,omitempty"`
}
type DB struct{ Licenses []License `json:"licenses"` }
var mu sync.Mutex
var db=DB{Licenses:[]License{}}
const dataFile="licenses.json"

type APIReq struct{ LicenseKey string `json:"licenseKey"`; DeviceID string `json:"deviceId"`; DeviceName string `json:"deviceName"`; Product string `json:"product"`; Version string `json:"version"` }
type APIResp struct{ OK bool `json:"ok"`; Status string `json:"status"`; Error string `json:"error,omitempty"`; Message string `json:"message,omitempty"`; Token string `json:"token,omitempty"`; LicenseHint string `json:"licenseHint,omitempty"`; ExpiresAt int64 `json:"expiresAt,omitempty"`; LicenseExpiresAt int64 `json:"licenseExpiresAt,omitempty"`; DurationDays int `json:"durationDays,omitempty"`; HeartbeatSeconds int `json:"heartbeatSeconds,omitempty"` }
type Claims struct{ Version int `json:"v"`; LicenseID string `json:"licenseId"`; Product string `json:"product"`; DeviceID string `json:"deviceId"`; IssuedAt int64 `json:"issuedAt"`; ExpiresAt int64 `json:"expiresAt"`; LicenseExpiresAt int64 `json:"licenseExpiresAt,omitempty"`; Revision int64 `json:"revision,omitempty"` }
type AdminCreate struct{ Note string `json:"note"`; Days int `json:"days"`; Count int `json:"count"` }
type AdminAction struct{ Key string `json:"key"`; Action string `json:"action"` }

func norm(s string)string{return strings.ToUpper(strings.TrimSpace(s))}
func nowRFC()string{return time.Now().UTC().Format(time.RFC3339)}
func allowedDays(d int)bool{return d==1||d==7||d==365}
func randHex(n int)string{b:=make([]byte,n);rand.Read(b);return strings.ToUpper(hex.EncodeToString(b))}
func randKey()string{const c="ABCDEFGHJKLMNPQRSTUVWXYZ23456789";rb:=make([]byte,16);rand.Read(rb);b:=make([]byte,16);for i:=range b{b[i]=c[int(rb[i])%len(c)]};return "MIST-"+string(b[:4])+"-"+string(b[4:8])+"-"+string(b[8:12])+"-"+string(b[12:16])}
func load(){if b,e:=os.ReadFile(dataFile);e==nil{_ = json.Unmarshal(b,&db)};for i:=range db.Licenses{if db.Licenses[i].Revision<=0{db.Licenses[i].Revision=1}}}
func save(){b,_:=json.MarshalIndent(db,"","  ");_ = os.WriteFile(dataFile,b,0600)}
func revoked(l *License,d string)bool{for _,x:=range l.RevokedDeviceIDs{if strings.EqualFold(x,d){return true}};return false}
func addRevoked(l *License,d string){d=norm(d);if d==""||revoked(l,d){return};l.RevokedDeviceIDs=append(l.RevokedDeviceIDs,d)}
func parseTime(v string)(time.Time,bool){t,e:=time.Parse(time.RFC3339,v);return t,e==nil}
func closeSession(l *License,when time.Time,reason string){if l.CurrentOnlineAt==""{return};st,ok:=parseTime(l.CurrentOnlineAt);if !ok{st=when};sec:=int64(when.Sub(st).Seconds());if sec<0{sec=0};l.Sessions=append([]Session{{OnlineAt:st.UTC().Format(time.RFC3339),OfflineAt:when.UTC().Format(time.RFC3339),DurationSeconds:sec,DeviceName:l.BoundDeviceName,EndReason:reason}},l.Sessions...);if len(l.Sessions)>500{l.Sessions=l.Sessions[:500]};l.LastOfflineAt=when.UTC().Format(time.RFC3339);l.CurrentOnlineAt="";l.LastHeartbeatAt=""}
func reconcile(l *License,now time.Time){if l.CurrentOnlineAt==""{return};hb,ok:=parseTime(l.LastHeartbeatAt);if !ok{hb,_=parseTime(l.CurrentOnlineAt)};if !hb.IsZero()&&now.Sub(hb)>=offlineAfter{closeSession(l,hb.Add(offlineAfter),"heartbeat_timeout")}}
func markOnline(l *License,now time.Time,name string){reconcile(l,now);if l.CurrentOnlineAt==""{l.CurrentOnlineAt=now.UTC().Format(time.RFC3339);l.LastOnlineAt=l.CurrentOnlineAt;l.OnlineCount++};l.LastHeartbeatAt=now.UTC().Format(time.RFC3339);l.LastSeenAt=l.LastHeartbeatAt;if name!=""{l.BoundDeviceName=name}}
func privateKey()(ed25519.PrivateKey,error){seed:=sha256.Sum256([]byte("EZQ_RENDER_FREE_SIGNING_KEY_V1"));return ed25519.NewKeyFromSeed(seed[:]),nil}
func sign(priv ed25519.PrivateKey,c Claims)(string,error){b,e:=json.Marshal(c);if e!=nil{return "",e};p:=base64.RawURLEncoding.EncodeToString(b);sig:=ed25519.Sign(priv,[]byte(p));return p+"."+base64.RawURLEncoding.EncodeToString(sig),nil}
func out(w http.ResponseWriter,code int,v any){w.Header().Set("Content-Type","application/json; charset=utf-8");w.Header().Set("Cache-Control","no-store");w.WriteHeader(code);_ = json.NewEncoder(w).Encode(v)}
func hint(k string)string{if len(k)<=10{return k};return k[:5]+"***"+k[len(k)-4:]}

func licenseHandler(mode string,priv ed25519.PrivateKey)http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){
 if r.Method!="POST"{out(w,405,APIResp{OK:false,Status:"method_not_allowed",Error:"POST required"});return}
 var q APIReq;b,_:=io.ReadAll(io.LimitReader(r.Body,1<<20));if json.Unmarshal(b,&q)!=nil{out(w,400,APIResp{OK:false,Status:"bad_request",Error:"请求格式错误"});return}
 q.LicenseKey=norm(q.LicenseKey);q.DeviceID=norm(q.DeviceID)
 if q.Product!=product{out(w,403,APIResp{OK:false,Status:"wrong_product",Error:"授权产品不匹配"});return}
 if q.LicenseKey==""||len(q.DeviceID)<32{out(w,400,APIResp{OK:false,Status:"bad_request",Error:"授权码或设备ID无效"});return}
 mu.Lock();defer mu.Unlock();idx:=-1;for i:=range db.Licenses{if db.Licenses[i].Key==q.LicenseKey{idx=i;break}}
 if idx<0{out(w,403,APIResp{OK:false,Status:"invalid_key",Error:"授权码不存在"});return}
 l:=&db.Licenses[idx];now:=time.Now().UTC();reconcile(l,now)
 if l.Status!="active"{save();out(w,403,APIResp{OK:false,Status:"disabled",Error:"授权已被管理员禁用",LicenseHint:hint(l.Key)});return}
 if l.ExpiresAt>0&&l.ExpiresAt<=now.Unix(){closeSession(l,now,"expired");save();out(w,403,APIResp{OK:false,Status:"expired",Error:"授权已到期",LicenseHint:hint(l.Key),LicenseExpiresAt:l.ExpiresAt,DurationDays:l.DurationDays});return}
 if mode=="activate"{
  if l.BoundDeviceID==""&&revoked(l,q.DeviceID){out(w,403,APIResp{OK:false,Status:"device_revoked",Error:"这台电脑已被管理员从该授权码解绑，不能再次使用此授权码"});return}
  if l.BoundDeviceID!=""&&!strings.EqualFold(l.BoundDeviceID,q.DeviceID){out(w,409,APIResp{OK:false,Status:"bound_other_device",Error:"此授权码已经绑定另一台电脑"});return}
  for i:=range db.Licenses{o:=&db.Licenses[i];if o.Key!=l.Key&&o.Status=="active"&&o.BoundDeviceID!=""&&strings.EqualFold(o.BoundDeviceID,q.DeviceID)&&(o.ExpiresAt<=0||o.ExpiresAt>now.Unix()){out(w,409,APIResp{OK:false,Status:"device_bound_other_license",Error:"这台电脑已经绑定另一个有效授权"});return}}
  if l.BoundDeviceID==""{l.BoundDeviceID=q.DeviceID;l.BoundDeviceName=q.DeviceName;if l.ActivatedAt==""{l.ActivatedAt=now.UTC().Format(time.RFC3339);if l.ExpiresAt==0&&l.DurationDays>0{l.ExpiresAt=now.Add(time.Duration(l.DurationDays)*24*time.Hour).Unix()}}}
 }else{
  if l.BoundDeviceID==""{out(w,403,APIResp{OK:false,Status:"server_unbound",Error:"该授权当前未绑定设备，请重新激活"});return}
  if !strings.EqualFold(l.BoundDeviceID,q.DeviceID){out(w,409,APIResp{OK:false,Status:"device_mismatch",Error:"授权绑定设备与当前电脑不一致"});return}
 }
 markOnline(l,now,q.DeviceName);save();exp:=now.Add(72*time.Hour).Unix();if l.ExpiresAt>0&&l.ExpiresAt<exp{exp=l.ExpiresAt}
 tok,e:=sign(priv,Claims{Version:1,LicenseID:l.ID,Product:product,DeviceID:q.DeviceID,IssuedAt:now.Unix(),ExpiresAt:exp,LicenseExpiresAt:l.ExpiresAt,Revision:l.Revision});if e!=nil{out(w,500,APIResp{OK:false,Status:"sign_error",Error:"授权签名失败"});return}
 out(w,200,APIResp{OK:true,Status:"authorized",Token:tok,LicenseHint:hint(l.Key),ExpiresAt:exp,LicenseExpiresAt:l.ExpiresAt,DurationDays:l.DurationDays,HeartbeatSeconds:60,Message:"授权有效"})
}}
func adminOK(r *http.Request)bool{u,p,ok:=r.BasicAuth();return ok&&u=="admin"&&p=="ezqadmin"}
func adminWrap(fn http.HandlerFunc)http.HandlerFunc{return func(w http.ResponseWriter,r *http.Request){if !adminOK(r){w.Header().Set("WWW-Authenticate","Basic");http.Error(w,"Unauthorized",401);return};fn(w,r)}}

func main(){
 priv,e:=privateKey();if e!=nil{log.Fatal(e)};load()
 mux:=http.NewServeMux()
 mux.HandleFunc("/health",func(w http.ResponseWriter,r *http.Request){out(w,200,map[string]any{"ok":true,"server":"MistLicenseServer","version":serverVersion})})
 mux.HandleFunc("/api/v1/activate",licenseHandler("activate",priv));mux.HandleFunc("/api/v1/validate",licenseHandler("validate",priv));mux.HandleFunc("/api/v1/heartbeat",licenseHandler("heartbeat",priv))
 mux.HandleFunc("/api/v1/admin/login",adminWrap(func(w http.ResponseWriter,r *http.Request){mu.Lock();defer mu.Unlock();out(w,200,map[string]any{"ok":true,"version":serverVersion,"summary":map[string]int{"total":len(db.Licenses)}})}))
 mux.HandleFunc("/api/v1/admin/licenses",adminWrap(func(w http.ResponseWriter,r *http.Request){mu.Lock();defer mu.Unlock();out(w,200,map[string]any{"ok":true,"version":serverVersion,"licenses":db.Licenses,"summary":map[string]int{"total":len(db.Licenses)}})}))
 mux.HandleFunc("/api/v1/admin/create",adminWrap(func(w http.ResponseWriter,r *http.Request){var q AdminCreate;_ = json.NewDecoder(io.LimitReader(r.Body,1<<20)).Decode(&q);if !allowedDays(q.Days){out(w,400,map[string]any{"ok":false,"error":"授权时长只允许 1天、1星期、1年"});return};if q.Count<=0{q.Count=1};if q.Count>100{q.Count=100};mu.Lock();defer mu.Unlock();created:=make([]License,0,q.Count);for i:=0;i<q.Count;i++{l:=License{ID:randHex(12),Key:randKey(),Status:"active",Note:strings.TrimSpace(q.Note),CreatedAt:nowRFC(),DurationDays:q.Days,Revision:1,Sessions:[]Session{}};db.Licenses=append([]License{l},db.Licenses...);created=append(created,l)};save();out(w,200,map[string]any{"ok":true,"version":serverVersion,"created":created,"message":"授权已生成"})}))
 mux.HandleFunc("/api/v1/admin/action",adminWrap(func(w http.ResponseWriter,r *http.Request){var q AdminAction;_ = json.NewDecoder(io.LimitReader(r.Body,1<<20)).Decode(&q);q.Key=norm(q.Key);mu.Lock();defer mu.Unlock();for i:=range db.Licenses{if db.Licenses[i].Key==q.Key{l:=&db.Licenses[i];now:=time.Now().UTC();switch q.Action{case"unbind":closeSession(l,now,"admin_unbind");addRevoked(l,l.BoundDeviceID);l.BoundDeviceID="";l.BoundDeviceName="";l.Revision++;case"disable":closeSession(l,now,"admin_disable");l.Status="disabled";l.Revision++;case"enable":l.Status="active";l.Revision++;default:out(w,400,map[string]any{"ok":false,"error":"unknown action"});return};save();out(w,200,map[string]any{"ok":true,"message":"操作成功"});return}};out(w,400,map[string]any{"ok":false,"error":"license not found"})}))
 go func(){t:=time.NewTicker(30*time.Second);for now:=range t.C{mu.Lock();for i:=range db.Licenses{reconcile(&db.Licenses[i],now.UTC())};save();mu.Unlock()}}()
 port:=os.Getenv("PORT");if port==""{port="10000"};log.Printf("EZQ license server listening on :%s",port);log.Fatal(http.ListenAndServe(":"+port,mux))
}

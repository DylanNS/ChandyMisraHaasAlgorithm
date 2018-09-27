package main
import (
"fmt"
"net"
"os"
"strconv"
"time"
"bufio"
"strings"
)
//Variáveis globais interessantes para o processo
var err string
var myPort string //porta do meu servidor
var nServers int //qtde de outros processo
var CliConn []*net.UDPConn //vetor com conexões para os servidores
 //dos outros processos
var ServConn *net.UDPConn //conexão do meu servidor (onde recebo
 //mensagens dos outros processos)
var id int //numero identificador do processo
var lastest []int
var wait []bool
var num  []int
var engager []int
var WF []int
var total int
var WF_len int

func CheckError(err1 error){
	if err1 != nil {
		fmt.Println("Erro: ", err1)
		os.Exit(0)
	}
}

func PrintError(err error) {
	if err != nil {
		fmt.Println("Erro: ", err)
	}
}

func send_reply(i int, m int,k int, j int){
	//construir msg
	msgstream := "R," + strconv.Itoa(i) + "," + strconv.Itoa(m) + "," + strconv.Itoa(k) + "," + strconv.Itoa(j)
	//encaminhar mensagem
	fmt.Printf("\nP%d: Enviou Reply(i = %d,m = %d,k = %d,j = %d) como %s para P%d\n",id,i,m,k,j,msgstream,j)
	buf := []byte(msgstream)
	to := j-1
     _,err := CliConn[to % nServers].Write(buf)
     if err != nil {
        fmt.Println(msgstream, err)
    }

}

func doServerJob() {

	 buf := make([]byte, 1024)
	 for {

		 n, _, err := ServConn.ReadFromUDP(buf)
		 //Tranformar para string 
		 //msg_received = "Q,i,m,j,k" ou "R,i,m,k,j"
		 msg_received := string (buf[0:n])
		 //Separa por virgula
		 //slice_msg = ["Q","i","m","j","k"] ou ["R","i","m","k","j"]
		 slice_msg := strings.Split(msg_received,",")
		 i, err := strconv.Atoi(slice_msg[1])
		 m,err := strconv.Atoi(slice_msg[2])
		 j,err := strconv.Atoi(slice_msg[3])
		 k,err := strconv.Atoi(slice_msg[4])
		//Pk ativo ignora
		if WF_len > 0 {
			if slice_msg[0] == "Q"{

				fmt.Printf("\nP%d Recebeu Query(i = %d,m = %d,j = %d, k =%d) como %s de P%d",id,i,m,j,k,msg_received,j)
				//query de Pi -> Pj -> Pk
				// Primeira Query ou Query de engajamento
				if (m > lastest[i]){
					//atualiza versao da detecção
					lastest[i] = m;
					//salva pai
					engager[i] = j;
					//Atualiza Espera
					wait[i] = true;
					//Total de denpendências esperadas
					num[i] = len(WF)
					//Envia Query
					send_query_2_all_in_WF(i,m,k)

				} else if wait[i] && m == lastest[i] {
					//Não é query engajadora,i.e., Query repetida
					//Manda Reply
					send_reply(i,m,k,j)
				}
			} else if slice_msg[0] == "R"{
				//Recebeu um Reply
				fmt.Printf("\nP%d Recebeu Reply(i = %d,m = %d,k = %d, j =%d) como %s de P%d",id,i,m,j,k,msg_received,j)
				if wait[i] && m==lastest[i]{
					num[i]--
					if num[i] == 0{
						//Recebeu todas as replies dependentes
						if i == id{
							//Foi quem iniciou a detecção 
							//Declara Deadlock
							fmt.Println("\nDeadLock Detectado!")
						} else {
							//Manda reply para o pai
							send_reply(i,m,k,engager[i])
						}
					}
				}
				

			}  else {
				fmt.Println("Error: mensagem estranha -> ",err,msg_received)
			}

			if err != nil {
				fmt.Println("Error: ",err)
			
			}
		 } 
	}
}

func doClientJob(otherProcess int,mymsg string) {
//Envia uma mensagem (string) para o servidor do processo id = (otherProcess)

     buf := []byte(mymsg)
     _,err := CliConn[otherProcess % nServers].Write(buf)
     if err != nil {
        fmt.Println(mymsg, err)
    }

}

func initConnections() {
	// Entrada: NomeDoComando,id,|WF|,portas[nServers+1],WFis
	id, _ = strconv.Atoi(os.Args[1])
	WF_len,_ = strconv.Atoi(os.Args[2])
	myPort = os.Args[ id + 2]
	nServers = len(os.Args) - 3 - WF_len
	total = nServers
	/*Esse 2 tira o nome (no caso Process),id e o total. As demais portas são dos outros processos*/
	connections := make([]*net.UDPConn, nServers, nServers)
	for i:=0; i<nServers; i++ {

		port := os.Args[i+3]
		
			ServerAddr,err := net.ResolveUDPAddr("udp","127.0.0.1" + string (port) )
			PrintError(err)
 
    		LocalAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
 		   	PrintError(err)
 
    		connections[i], err = net.DialUDP("udp",LocalAddr, ServerAddr)
			PrintError(err)
	}
	for ind:=0; ind < WF_len ; ind ++{
		wfi,_ := strconv.Atoi(os.Args[ind+nServers+3])
		WF = append(WF,wfi)
	} 

	for index:=0; index < total; index++{
		lastest = append(lastest,0)
		engager = append(engager,0)
		num = append(num,0)
		wait = append(wait,false)
	}

	CliConn = connections

	 /* Inicializando Servidor no myPort*/   
	 ServerAddr,err := net.ResolveUDPAddr("udp", myPort)
	 CheckError(err)
	
	 /* Ouvindo na minha porta*/
	 ServConn, err = net.ListenUDP("udp", ServerAddr)
	 CheckError(err)

}

func readInput(ch chan string) {
	// Non-blocking async routine to listen for terminal input
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _, _ := reader.ReadLine()
		ch <- string(text)
	}
}

func send_query_2_all_in_WF(i int, m int, j int){

	//construir msg
	msgstream := "Q," + strconv.Itoa(i) + "," + strconv.Itoa(m) + "," + strconv.Itoa(j) + ","
	//encaminhar mensagem
	for _,k := range WF {
		mymsg := msgstream+strconv.Itoa(k)
		fmt.Printf("\nP%d: Enviou Query(i = %d,m = %d,j = %d,k = %d) como %s para P%d\n",id,i,m,j,k,mymsg,k)
		doClientJob(k-1,mymsg)
	}
}


func star_from_Pi(){

	fmt.Printf("\n Inicio da Deteccao P%d",id)
	lastest[id]++
	m := lastest[id]
	wait[id] = true
	num[id] = WF_len
	send_query_2_all_in_WF(id,m,id)

}

func main(){
	initConnections()
	//O fechamento de conexões devem ficar aqui, assim só fecha
	//conexão quando a main morrer
	defer ServConn.Close()
	for i := 0; i < nServers; i++ {
		defer CliConn[i].Close()
	}
	//Todo Process fará a mesma coisa: ouvir msg e mandar infinitos
	//i’s para os outros processos
	ch := make(chan string)
	go readInput(ch)
	for {
			
		//Server
		go doServerJob()
		//Atendendo input do teclado
		select {
			case x, valid := <- ch :
				if valid {
						if ( strings.ToLower(x) == "start" && WF_len > 0 ){
							star_from_Pi() //inicia a detecção do deadlock.
						}
				} else {
						 
					fmt.Println("Channel closed!")
						
				}
				
			default:
			
				// Do nothing in the non-blocking approach.
			
				time.Sleep(time.Second * 1)
		}
			
		// Wait a while
		time.Sleep(time.Second * 1)
	}
}
		
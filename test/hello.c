#include <systemd/sd-journal.h>
#include <unistd.h>
#include <stdlib.h>

// Emit some sample message
int main(int argc, char *argv[]) {
    sd_journal_send("MESSAGE=Hello World!",
                    "MESSAGE_ID=52fb62f99e2c49d89cfbf9d6de5e3555",
                    "PRIORITY=5",
                    "HOME=%s", getenv("HOME"),
                    "TERM=%s", getenv("TERM"),
                    "PAGE_SIZE=%li", sysconf(_SC_PAGESIZE),
                    "N_CPUS=%li", sysconf(_SC_NPROCESSORS_ONLN),
                    NULL);
    return 0;
}

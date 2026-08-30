import subprocess
import os
import urllib.request
import tarfile
import sys

def build_and_run():
    # 1. Download and extract Go if not already built
    if not os.path.exists("./server"):
        print("Go server binary not found. Preparing to build...")
        
        # Download Go
        go_tar = "/tmp/go.tar.gz"
        go_url = "https://go.dev/dl/go1.22.2.linux-amd64.tar.gz"
        print(f"Downloading Go compiler from {go_url}...")
        urllib.request.urlretrieve(go_url, go_tar)
        
        print("Extracting Go...")
        with tarfile.open(go_tar, "r:gz") as tar:
            tar.extractall(path="/tmp")
            
        print("Building Go backend...")
        # Compile Go server using extracted Go compiler
        env = os.environ.copy()
        env["PATH"] = f"/tmp/go/bin:{env.get('PATH', '')}"
        env["CGO_ENABLED"] = "0"
        
        build_proc = subprocess.run(
            ["/tmp/go/bin/go", "build", "-o", "server", "."],
            cwd="./backend",
            env=env,
            capture_output=True,
            text=True
        )
        
        if build_proc.returncode != 0:
            print("Go build failed!")
            print("STDOUT:", build_proc.stdout)
            print("STDERR:", build_proc.stderr)
            sys.exit(1)
            
        os.rename("./backend/server", "./server")
        print("Go backend built successfully!")

    # 2. Run the Go server
    print("Starting Go server...")
    port = os.environ.get("PORT", "7860")
    env = os.environ.copy()
    env["BACKEND_PORT"] = port
    
    server_proc = subprocess.Popen(
        ["./server"],
        env=env,
        stdout=sys.stdout,
        stderr=sys.stderr
    )
    
    server_proc.wait()

if __name__ == "__main__":
    build_and_run()

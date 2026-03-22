import sys
from yaml import dump
def generate_file_compose(output, clients):
        
    sections = {}
    services = {} 
    sections['name'] = 'tp0'
    
    services['server'] = {
        'container_name':'server',
        'image':'server:latest',
        'entrypoint':'/server',
        'networks':['testing_net'],
        'volumes':['./server/config.ini:/config.ini']
    }
    for i in range(1,clients+1):
        client = f'client{i}'
        services[client] = {
            'container_name': client,
            'image':'client:latest',
            'entrypoint':'/client',
            'environment': [f'CLI_ID={i}'],
            'networks':['testing_net'],
            'depends_on':['server'],
            'volumes':['./client/config.yaml:/config.yaml']
        }
    sections['services'] = services
    sections['networks'] = {
        'testing_net': {
            'ipam':{
                'driver':'default',
                'config': [{'subnet': '172.25.125.0/24'}]
            }
        }
    }
    with open(output,'w') as file:
        dump(sections, file, sort_keys=False)



def main():

    if len(sys.argv) != 3:
        print("expected 2 params: output_file_name; number_clients")
        return
    output_file_name = sys.argv[1] 
    if not (output_file_name.endswith('.yaml')):
        print("output_file_name must contain .yaml extension")
        return
    try:
        clients = int(sys.argv[2])
    except ValueError:
        print("No se puede convertir el segundo parametro a numero, asegurate de que sea un numero entero")
        return

    generate_file_compose(output_file_name, clients)

if __name__ == "__main__":
    main()
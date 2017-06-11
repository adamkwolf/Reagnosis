/* Djikstra from
http://rosettacode.org/wiki/Dijkstra%27s_algorithm#C
*/

/* The graph corresponds to a simple room layout with

+++++++++++++++++++++++++++
+    +     +          +   + 
+ 0  +  12 +   13     +   +
+ |  +  |  +   |      +   +
++|+++++|++++++| ++++++   +
+ 1---4-5------6----7---8 +
++|+++|++++++++++++ | +   +
+ | + |  +   +    + | +   +
+ | + |  +   +    + | +   + 
+ 2---3---11---10---9 +   +
+   +   +    +    +   +   +
+   +   +    +    +   +   +
+++++++++++++++++++++++++++


*/

#include <stdio.h>
#include <stdlib.h>
#include <limits.h>
 
typedef struct {
    int vertex;
    int weight;
} edge_t;
 
typedef struct {
    edge_t **edges;
    int edges_len;
    int edges_size;
    int dist;
    int prev;
    int visited;
} vertex_t;
 
typedef struct {
    vertex_t **vertices;
    int vertices_len;
    int vertices_size;
} graph_t;
 
typedef struct {
    int *data;
    int *prio;
    int *index;
    int len;
    int size;
} heap_t;
 
void add_vertex (graph_t *g, int i) {
    if (g->vertices_size < i + 1) {
        int size = g->vertices_size * 2 > i ? g->vertices_size * 2 : i + 4;
        g->vertices = realloc(g->vertices, size * sizeof (vertex_t *));
        for (int j = g->vertices_size; j < size; j++)
            g->vertices[j] = NULL;
        g->vertices_size = size;
    }
    if (!g->vertices[i]) {
        g->vertices[i] = calloc(1, sizeof (vertex_t));
        g->vertices_len++;
    }
}
 
void add_edge (graph_t *g, int a, int b, int w) {
    //a = a - 'a';
    //b = b - 'a';
    add_vertex(g, a);
    add_vertex(g, b);
    vertex_t *v = g->vertices[a];
    if (v->edges_len >= v->edges_size) {
        v->edges_size = v->edges_size ? v->edges_size * 2 : 4;
        v->edges = realloc(v->edges, v->edges_size * sizeof (edge_t *));
    }
    edge_t *e = calloc(1, sizeof (edge_t));
    e->vertex = b;
    e->weight = w;
    v->edges[v->edges_len++] = e;
}
 
heap_t *create_heap (int n) {
    heap_t *h = calloc(1, sizeof (heap_t));
    h->data = calloc(n + 1, sizeof (int));
    h->prio = calloc(n + 1, sizeof (int));
    h->index = calloc(n, sizeof (int));
    return h;
}
 
void push_heap (heap_t *h, int v, int p) {
    int i = h->index[v] == 0 ? ++h->len : h->index[v];
    int j = i / 2;
    while (i > 1) {
        if (h->prio[j] < p)
            break;
        h->data[i] = h->data[j];
        h->prio[i] = h->prio[j];
        h->index[h->data[i]] = i;
        i = j;
        j = j / 2;
    }
    h->data[i] = v;
    h->prio[i] = p;
    h->index[v] = i;
}
 
int min (heap_t *h, int i, int j, int k) {
    int m = i;
    if (j <= h->len && h->prio[j] < h->prio[m])
        m = j;
    if (k <= h->len && h->prio[k] < h->prio[m])
        m = k;
    return m;
}
 
int pop_heap (heap_t *h) {
    int v = h->data[1];
    int i = 1;
    while (1) {
        int j = min(h, h->len, 2 * i, 2 * i + 1);
        if (j == h->len)
            break;
        h->data[i] = h->data[j];
        h->prio[i] = h->prio[j];
        h->index[h->data[i]] = i;
        i = j;
    }
    h->data[i] = h->data[h->len];
    h->prio[i] = h->prio[h->len];
    h->index[h->data[i]] = i;
    h->len--;
    return v;
}
 
void dijkstra (graph_t *g, int a, int b) {
    int i, j;
    for (i = 0; i < g->vertices_len; i++) {
        vertex_t *v = g->vertices[i];
        v->dist = INT_MAX;
        v->prev = 0;
        v->visited = 0;
    }
    vertex_t *v = g->vertices[a];
    v->dist = 0;
    heap_t *h = create_heap(g->vertices_len);
    push_heap(h, a, v->dist);
    while (h->len) {
        i = pop_heap(h);
        if (i == b)
            break;
        v = g->vertices[i];
        v->visited = 1;
        for (j = 0; j < v->edges_len; j++) {
            edge_t *e = v->edges[j];
            vertex_t *u = g->vertices[e->vertex];
            if (!u->visited && v->dist + e->weight <= u->dist) {
                u->prev = i;
                u->dist = v->dist + e->weight;
                push_heap(h, e->vertex, u->dist);
            }
        }
    }
}
 
void print_path (graph_t *g, int i) {
    int n, j, k;
    vertex_t *v, *u;
    char floorplan[13][27]=
      {{'+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+'},
       {'+',' ',' ',' ',' ','+',' ',' ',' ',' ',' ','+',' ',' ',' ',' ',' ',' ',' ',' ',' ',' ','+',' ',' ',' ','+'},
       {'+',' ','X',' ',' ','+',' ',' ','X',' ',' ','+',' ',' ',' ','X',' ',' ',' ',' ',' ',' ','+',' ',' ',' ','+'},
       {'+',' ','|',' ',' ','+',' ',' ','|',' ',' ','+',' ',' ',' ','|',' ',' ',' ',' ',' ',' ','+',' ',' ',' ','+'},
       {'+','+','|','+','+','+','+','+','|','+','+','+','+','+','+','|',' ','+','+','+','+','+','+',' ',' ',' ','+'},
       {'+',' ','X','-','-','-','X','-','X','-','-','-','-','-','-','X','-','-','-','-','X','-','-','-','X',' ','+'},
       {'+','+','|','+','+','+','|','+','+','+','+','+','+','+','+','+','+','+','+','+','|',' ','+',' ',' ',' ','+'},
       {'+',' ','|',' ','+',' ','|',' ',' ','+',' ',' ',' ','+',' ',' ',' ',' ','+',' ','|',' ','+',' ',' ',' ','+'},
       {'+',' ','|',' ','+',' ','|',' ',' ','+',' ',' ',' ','+',' ',' ',' ',' ','+',' ','|',' ','+',' ',' ',' ','+'},
       {'+',' ','X','-','-','-','X','-','-','-','X','-','-','-','-','X','-','-','-','-','X',' ','+',' ',' ',' ','+'},
       {'+',' ',' ',' ','+',' ',' ',' ','+',' ',' ',' ',' ','+',' ',' ',' ',' ','+',' ',' ',' ','+',' ',' ',' ','+'},
       {'+',' ',' ',' ','+',' ',' ',' ','+',' ',' ',' ',' ','+',' ',' ',' ',' ','+',' ',' ',' ','+',' ',' ',' ','+'},
       {'+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+','+'}};



    v = g->vertices[i];
    if (v->dist == INT_MAX) {
        printf("no path\n");
        return;
    }
    
    for (n = 1, u = v; u->dist; u = g->vertices[u->prev], n++)
        ;
    int *path = malloc(n);
    path[n - 1] =  i;
    for (j = 0, u = v; u->dist; u = g->vertices[u->prev], j++)
    {
        path[n - j - 2] = u->prev;
        printf("%d\n",path[n - j - 2]);
        if(path[n-j-2]==0)  floorplan[2][2]='0';
        if(path[n-j-2]==1)  floorplan[5][2]='1';
        if(path[n-j-2]==2)  floorplan[9][2]='2';
        if(path[n-j-2]==3)  floorplan[9][6]='3';
        if(path[n-j-2]==4)  floorplan[5][6]='4';
        if(path[n-j-2]==5)  floorplan[5][8]='5';
        if(path[n-j-2]==6)  floorplan[5][15]='6';
        if(path[n-j-2]==7)  floorplan[5][20]='7';
        if(path[n-j-2]==8)  floorplan[5][23]='8';
        if(path[n-j-2]==9)  floorplan[9][6]='9';
        if(path[n-j-2]==10) {
                             floorplan[9][8]='1';
                             floorplan[9][9]='0';
                            }
        if(path[n-j-2]==11) {
                            floorplan[9][14]='1';
                            floorplan[9][15]='1';
                            }
        if(path[n-j-2]==12) {
                             floorplan[2][8]='1';
                             floorplan[2][9]='2';
                             }
        if(path[n-j-2]==13) {
                             floorplan[2][15]='1';
                             floorplan[2][16]='3';
                             }

    }
    printf("%d  \n", v->dist);
    for(k=0;k<13;k++)
    {
       for(j=0;j<27;j++) printf("%c",floorplan[k][j]);
       printf("\n");
    }
}
 
int main () {
    graph_t *g = calloc(1, sizeof (graph_t));
    add_edge(g, 0, 1, 1);
    add_edge(g, 1, 2, 1);
    add_edge(g, 2, 3, 1);
    add_edge(g, 3, 4, 1);
    add_edge(g, 4, 5, 1);
    add_edge(g, 5, 6, 1);
    add_edge(g, 6, 7, 1);
    add_edge(g, 7, 8, 1);
    add_edge(g, 7, 9, 1);
    add_edge(g, 9, 10, 1);
    add_edge(g, 10, 11, 1);
    add_edge(g, 11, 3, 1);
    add_edge(g, 5, 12, 1);
    add_edge(g, 6, 13, 1);

    dijkstra(g, 0, 8);
    print_path(g, 9);
    return 0;
}

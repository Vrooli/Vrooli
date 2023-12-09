FROM squidfunk/mkdocs-material:9.3

# Macros plugin required to use variables in markdown
RUN pip install mkdocs-macros-plugin
# RUN pip install neoteroi-mkdocs # Old OpenAPI spec displayer. Doesn't display nullable relations
# RUN pip install mkdocs-render-swagger-plugin # Looks better than previous, but can crash page when expanding a type with circular reference
# Current option identical to mkdocs-render-swagger-plugin. Also has the circular reference issue
RUN pip install mkdocs-swagger-ui-tag
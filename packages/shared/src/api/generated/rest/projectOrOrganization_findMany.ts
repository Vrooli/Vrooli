export const projectOrOrganization_findMany = {
  "fieldName": "projectOrOrganizations",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrOrganizations",
        "loc": {
          "start": 1985,
          "end": 2007
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2008,
              "end": 2013
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2016,
                "end": 2021
              }
            },
            "loc": {
              "start": 2015,
              "end": 2021
            }
          },
          "loc": {
            "start": 2008,
            "end": 2021
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 2029,
                "end": 2034
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 2045,
                      "end": 2051
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2045,
                    "end": 2051
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2060,
                      "end": 2064
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 2086,
                              "end": 2093
                            }
                          },
                          "loc": {
                            "start": 2086,
                            "end": 2093
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 2115,
                                  "end": 2127
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2112,
                                "end": 2127
                              }
                            }
                          ],
                          "loc": {
                            "start": 2094,
                            "end": 2141
                          }
                        },
                        "loc": {
                          "start": 2079,
                          "end": 2141
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Organization",
                            "loc": {
                              "start": 2161,
                              "end": 2173
                            }
                          },
                          "loc": {
                            "start": 2161,
                            "end": 2173
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Organization_list",
                                "loc": {
                                  "start": 2195,
                                  "end": 2212
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2192,
                                "end": 2212
                              }
                            }
                          ],
                          "loc": {
                            "start": 2174,
                            "end": 2226
                          }
                        },
                        "loc": {
                          "start": 2154,
                          "end": 2226
                        }
                      }
                    ],
                    "loc": {
                      "start": 2065,
                      "end": 2236
                    }
                  },
                  "loc": {
                    "start": 2060,
                    "end": 2236
                  }
                }
              ],
              "loc": {
                "start": 2035,
                "end": 2242
              }
            },
            "loc": {
              "start": 2029,
              "end": 2242
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2247,
                "end": 2255
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 2266,
                      "end": 2277
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2266,
                    "end": 2277
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 2286,
                      "end": 2302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2286,
                    "end": 2302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorOrganization",
                    "loc": {
                      "start": 2311,
                      "end": 2332
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2311,
                    "end": 2332
                  }
                }
              ],
              "loc": {
                "start": 2256,
                "end": 2338
              }
            },
            "loc": {
              "start": 2247,
              "end": 2338
            }
          }
        ],
        "loc": {
          "start": 2023,
          "end": 2342
        }
      },
      "loc": {
        "start": 1985,
        "end": 2342
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 9,
          "end": 19
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 23,
            "end": 28
          }
        },
        "loc": {
          "start": 23,
          "end": 28
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 31,
                "end": 33
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 31,
              "end": 33
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 34,
                "end": 44
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 34,
              "end": 44
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 45,
                "end": 55
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 45,
              "end": 55
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 56,
                "end": 61
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 56,
              "end": 61
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 62,
                "end": 67
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 62,
              "end": 67
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 68,
                "end": 73
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 87,
                        "end": 99
                      }
                    },
                    "loc": {
                      "start": 87,
                      "end": 99
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 113,
                            "end": 129
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 110,
                          "end": 129
                        }
                      }
                    ],
                    "loc": {
                      "start": 100,
                      "end": 135
                    }
                  },
                  "loc": {
                    "start": 80,
                    "end": 135
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 147,
                        "end": 151
                      }
                    },
                    "loc": {
                      "start": 147,
                      "end": 151
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 165,
                            "end": 173
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 162,
                          "end": 173
                        }
                      }
                    ],
                    "loc": {
                      "start": 152,
                      "end": 179
                    }
                  },
                  "loc": {
                    "start": 140,
                    "end": 179
                  }
                }
              ],
              "loc": {
                "start": 74,
                "end": 181
              }
            },
            "loc": {
              "start": 68,
              "end": 181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 182,
                "end": 185
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 192,
                      "end": 201
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 192,
                    "end": 201
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 206,
                      "end": 215
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 206,
                    "end": 215
                  }
                }
              ],
              "loc": {
                "start": 186,
                "end": 217
              }
            },
            "loc": {
              "start": 182,
              "end": 217
            }
          }
        ],
        "loc": {
          "start": 29,
          "end": 219
        }
      },
      "loc": {
        "start": 0,
        "end": 219
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 229,
          "end": 246
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 250,
            "end": 262
          }
        },
        "loc": {
          "start": 250,
          "end": 262
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 265,
                "end": 267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 265,
              "end": 267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 268,
                "end": 274
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 268,
              "end": 274
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 275,
                "end": 285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 275,
              "end": 285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 286,
                "end": 296
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 286,
              "end": 296
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 297,
                "end": 315
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 297,
              "end": 315
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 316,
                "end": 325
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 316,
              "end": 325
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 326,
                "end": 339
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 326,
              "end": 339
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 340,
                "end": 352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 340,
              "end": 352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 353,
                "end": 365
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 353,
              "end": 365
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 366,
                "end": 375
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 366,
              "end": 375
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 376,
                "end": 380
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 390,
                      "end": 398
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 387,
                    "end": 398
                  }
                }
              ],
              "loc": {
                "start": 381,
                "end": 400
              }
            },
            "loc": {
              "start": 376,
              "end": 400
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 401,
                "end": 413
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 420,
                      "end": 422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 420,
                    "end": 422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 427,
                      "end": 435
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 427,
                    "end": 435
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 440,
                      "end": 443
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 440,
                    "end": 443
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 448,
                      "end": 452
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 448,
                    "end": 452
                  }
                }
              ],
              "loc": {
                "start": 414,
                "end": 454
              }
            },
            "loc": {
              "start": 401,
              "end": 454
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 455,
                "end": 458
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 465,
                      "end": 478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 465,
                    "end": 478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 483,
                      "end": 492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 483,
                    "end": 492
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 497,
                      "end": 508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 497,
                    "end": 508
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 513,
                      "end": 522
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 513,
                    "end": 522
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 527,
                      "end": 536
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 527,
                    "end": 536
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 541,
                      "end": 548
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 541,
                    "end": 548
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 553,
                      "end": 565
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 553,
                    "end": 565
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 570,
                      "end": 578
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 570,
                    "end": 578
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 583,
                      "end": 597
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 608,
                            "end": 610
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 608,
                          "end": 610
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 619,
                            "end": 629
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 619,
                          "end": 629
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 638,
                            "end": 648
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 638,
                          "end": 648
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 657,
                            "end": 664
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 657,
                          "end": 664
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 673,
                            "end": 684
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 673,
                          "end": 684
                        }
                      }
                    ],
                    "loc": {
                      "start": 598,
                      "end": 690
                    }
                  },
                  "loc": {
                    "start": 583,
                    "end": 690
                  }
                }
              ],
              "loc": {
                "start": 459,
                "end": 692
              }
            },
            "loc": {
              "start": 455,
              "end": 692
            }
          }
        ],
        "loc": {
          "start": 263,
          "end": 694
        }
      },
      "loc": {
        "start": 220,
        "end": 694
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 704,
          "end": 720
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 724,
            "end": 736
          }
        },
        "loc": {
          "start": 724,
          "end": 736
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 739,
                "end": 741
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 739,
              "end": 741
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 742,
                "end": 748
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 742,
              "end": 748
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 749,
                "end": 752
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 759,
                      "end": 772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 759,
                    "end": 772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 777,
                      "end": 786
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 786
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 791,
                      "end": 802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 791,
                    "end": 802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 807,
                      "end": 816
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 807,
                    "end": 816
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 821,
                      "end": 830
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 821,
                    "end": 830
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 835,
                      "end": 842
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 835,
                    "end": 842
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 847,
                      "end": 859
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 847,
                    "end": 859
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 864,
                      "end": 872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 864,
                    "end": 872
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 877,
                      "end": 891
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 902,
                            "end": 904
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 902,
                          "end": 904
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 913,
                            "end": 923
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 913,
                          "end": 923
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 932,
                            "end": 942
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 932,
                          "end": 942
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 951,
                            "end": 958
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 951,
                          "end": 958
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 967,
                            "end": 978
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 967,
                          "end": 978
                        }
                      }
                    ],
                    "loc": {
                      "start": 892,
                      "end": 984
                    }
                  },
                  "loc": {
                    "start": 877,
                    "end": 984
                  }
                }
              ],
              "loc": {
                "start": 753,
                "end": 986
              }
            },
            "loc": {
              "start": 749,
              "end": 986
            }
          }
        ],
        "loc": {
          "start": 737,
          "end": 988
        }
      },
      "loc": {
        "start": 695,
        "end": 988
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 998,
          "end": 1010
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 1014,
            "end": 1021
          }
        },
        "loc": {
          "start": 1014,
          "end": 1021
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1024,
                "end": 1032
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1039,
                      "end": 1051
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1062,
                            "end": 1064
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1062,
                          "end": 1064
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1073,
                            "end": 1081
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1073,
                          "end": 1081
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1090,
                            "end": 1101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1090,
                          "end": 1101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1110,
                            "end": 1114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1110,
                          "end": 1114
                        }
                      }
                    ],
                    "loc": {
                      "start": 1052,
                      "end": 1120
                    }
                  },
                  "loc": {
                    "start": 1039,
                    "end": 1120
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1125,
                      "end": 1127
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1125,
                    "end": 1127
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1132,
                      "end": 1142
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1132,
                    "end": 1142
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1147,
                      "end": 1157
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1147,
                    "end": 1157
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 1162,
                      "end": 1178
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1162,
                    "end": 1178
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1183,
                      "end": 1191
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1183,
                    "end": 1191
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1196,
                      "end": 1205
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1196,
                    "end": 1205
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1210,
                      "end": 1222
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1210,
                    "end": 1222
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 1227,
                      "end": 1243
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1227,
                    "end": 1243
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1248,
                      "end": 1258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1248,
                    "end": 1258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1263,
                      "end": 1275
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1263,
                    "end": 1275
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1280,
                      "end": 1292
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1280,
                    "end": 1292
                  }
                }
              ],
              "loc": {
                "start": 1033,
                "end": 1294
              }
            },
            "loc": {
              "start": 1024,
              "end": 1294
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1295,
                "end": 1297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1295,
              "end": 1297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1298,
                "end": 1308
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1298,
              "end": 1308
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1309,
                "end": 1319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1309,
              "end": 1319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1320,
                "end": 1329
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1320,
              "end": 1329
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1330,
                "end": 1341
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1330,
              "end": 1341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1342,
                "end": 1348
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1358,
                      "end": 1368
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1355,
                    "end": 1368
                  }
                }
              ],
              "loc": {
                "start": 1349,
                "end": 1370
              }
            },
            "loc": {
              "start": 1342,
              "end": 1370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1371,
                "end": 1376
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1390,
                        "end": 1402
                      }
                    },
                    "loc": {
                      "start": 1390,
                      "end": 1402
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1416,
                            "end": 1432
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1413,
                          "end": 1432
                        }
                      }
                    ],
                    "loc": {
                      "start": 1403,
                      "end": 1438
                    }
                  },
                  "loc": {
                    "start": 1383,
                    "end": 1438
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1450,
                        "end": 1454
                      }
                    },
                    "loc": {
                      "start": 1450,
                      "end": 1454
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1468,
                            "end": 1476
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1465,
                          "end": 1476
                        }
                      }
                    ],
                    "loc": {
                      "start": 1455,
                      "end": 1482
                    }
                  },
                  "loc": {
                    "start": 1443,
                    "end": 1482
                  }
                }
              ],
              "loc": {
                "start": 1377,
                "end": 1484
              }
            },
            "loc": {
              "start": 1371,
              "end": 1484
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1485,
                "end": 1496
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1485,
              "end": 1496
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1497,
                "end": 1511
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1497,
              "end": 1511
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1512,
                "end": 1517
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1512,
              "end": 1517
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1518,
                "end": 1527
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1518,
              "end": 1527
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1528,
                "end": 1532
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1542,
                      "end": 1550
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1539,
                    "end": 1550
                  }
                }
              ],
              "loc": {
                "start": 1533,
                "end": 1552
              }
            },
            "loc": {
              "start": 1528,
              "end": 1552
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1553,
                "end": 1567
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1553,
              "end": 1567
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1568,
                "end": 1573
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1568,
              "end": 1573
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1574,
                "end": 1577
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1584,
                      "end": 1593
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1584,
                    "end": 1593
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1598,
                      "end": 1609
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1598,
                    "end": 1609
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1614,
                      "end": 1625
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1614,
                    "end": 1625
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1630,
                      "end": 1639
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1630,
                    "end": 1639
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1644,
                      "end": 1651
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1644,
                    "end": 1651
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1656,
                      "end": 1664
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1656,
                    "end": 1664
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1669,
                      "end": 1681
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1669,
                    "end": 1681
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1686,
                      "end": 1694
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1686,
                    "end": 1694
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1699,
                      "end": 1707
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1699,
                    "end": 1707
                  }
                }
              ],
              "loc": {
                "start": 1578,
                "end": 1709
              }
            },
            "loc": {
              "start": 1574,
              "end": 1709
            }
          }
        ],
        "loc": {
          "start": 1022,
          "end": 1711
        }
      },
      "loc": {
        "start": 989,
        "end": 1711
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 1721,
          "end": 1729
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 1733,
            "end": 1736
          }
        },
        "loc": {
          "start": 1733,
          "end": 1736
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1739,
                "end": 1741
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1739,
              "end": 1741
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1742,
                "end": 1752
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1742,
              "end": 1752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 1753,
                "end": 1756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1753,
              "end": 1756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1757,
                "end": 1766
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1757,
              "end": 1766
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1767,
                "end": 1779
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1786,
                      "end": 1788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1786,
                    "end": 1788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1793,
                      "end": 1801
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1793,
                    "end": 1801
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1806,
                      "end": 1817
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1806,
                    "end": 1817
                  }
                }
              ],
              "loc": {
                "start": 1780,
                "end": 1819
              }
            },
            "loc": {
              "start": 1767,
              "end": 1819
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1820,
                "end": 1823
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 1830,
                      "end": 1835
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1830,
                    "end": 1835
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1840,
                      "end": 1852
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1840,
                    "end": 1852
                  }
                }
              ],
              "loc": {
                "start": 1824,
                "end": 1854
              }
            },
            "loc": {
              "start": 1820,
              "end": 1854
            }
          }
        ],
        "loc": {
          "start": 1737,
          "end": 1856
        }
      },
      "loc": {
        "start": 1712,
        "end": 1856
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 1866,
          "end": 1874
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 1878,
            "end": 1882
          }
        },
        "loc": {
          "start": 1878,
          "end": 1882
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1885,
                "end": 1887
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1885,
              "end": 1887
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 1888,
                "end": 1893
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1888,
              "end": 1893
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1894,
                "end": 1898
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1894,
              "end": 1898
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1899,
                "end": 1905
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1899,
              "end": 1905
            }
          }
        ],
        "loc": {
          "start": 1883,
          "end": 1907
        }
      },
      "loc": {
        "start": 1857,
        "end": 1907
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "projectOrOrganizations",
      "loc": {
        "start": 1915,
        "end": 1937
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 1939,
              "end": 1944
            }
          },
          "loc": {
            "start": 1938,
            "end": 1944
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ProjectOrOrganizationSearchInput",
              "loc": {
                "start": 1946,
                "end": 1978
              }
            },
            "loc": {
              "start": 1946,
              "end": 1978
            }
          },
          "loc": {
            "start": 1946,
            "end": 1979
          }
        },
        "directives": [],
        "loc": {
          "start": 1938,
          "end": 1979
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "projectOrOrganizations",
            "loc": {
              "start": 1985,
              "end": 2007
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2008,
                  "end": 2013
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2016,
                    "end": 2021
                  }
                },
                "loc": {
                  "start": 2015,
                  "end": 2021
                }
              },
              "loc": {
                "start": 2008,
                "end": 2021
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 2029,
                    "end": 2034
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 2045,
                          "end": 2051
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2045,
                        "end": 2051
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2060,
                          "end": 2064
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 2086,
                                  "end": 2093
                                }
                              },
                              "loc": {
                                "start": 2086,
                                "end": 2093
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 2115,
                                      "end": 2127
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2112,
                                    "end": 2127
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2094,
                                "end": 2141
                              }
                            },
                            "loc": {
                              "start": 2079,
                              "end": 2141
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Organization",
                                "loc": {
                                  "start": 2161,
                                  "end": 2173
                                }
                              },
                              "loc": {
                                "start": 2161,
                                "end": 2173
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Organization_list",
                                    "loc": {
                                      "start": 2195,
                                      "end": 2212
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2192,
                                    "end": 2212
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2174,
                                "end": 2226
                              }
                            },
                            "loc": {
                              "start": 2154,
                              "end": 2226
                            }
                          }
                        ],
                        "loc": {
                          "start": 2065,
                          "end": 2236
                        }
                      },
                      "loc": {
                        "start": 2060,
                        "end": 2236
                      }
                    }
                  ],
                  "loc": {
                    "start": 2035,
                    "end": 2242
                  }
                },
                "loc": {
                  "start": 2029,
                  "end": 2242
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2247,
                    "end": 2255
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 2266,
                          "end": 2277
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2266,
                        "end": 2277
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 2286,
                          "end": 2302
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2286,
                        "end": 2302
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorOrganization",
                        "loc": {
                          "start": 2311,
                          "end": 2332
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2311,
                        "end": 2332
                      }
                    }
                  ],
                  "loc": {
                    "start": 2256,
                    "end": 2338
                  }
                },
                "loc": {
                  "start": 2247,
                  "end": 2338
                }
              }
            ],
            "loc": {
              "start": 2023,
              "end": 2342
            }
          },
          "loc": {
            "start": 1985,
            "end": 2342
          }
        }
      ],
      "loc": {
        "start": 1981,
        "end": 2344
      }
    },
    "loc": {
      "start": 1909,
      "end": 2344
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrOrganization_findMany"
  }
};
